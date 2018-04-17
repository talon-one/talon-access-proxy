package dnscache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

type cacheEntry struct {
	OriginServer    string
	OriginServerNet string
	ValidUntil      time.Time
	RR              dns.RR
	Refreshing      bool
}

// DNSCache is a net.Resolver compliant DNS Resolver
type DNSCache struct {
	Logger            *zap.Logger
	MaxLookupAttempts int
	DialTimeout       time.Duration
	entries           []cacheEntry
	server            dns.Server
	mu                sync.Mutex
	closed            bool
	localAddr         *net.UDPAddr
}

// New creates a new DNSCache
func New(logger *zap.Logger) *DNSCache {
	return &DNSCache{
		Logger:            logger,
		MaxLookupAttempts: 4,
		DialTimeout:       time.Second * 30,
		closed:            true,
	}
}

// Server starts a DNSCache server
func (cache *DNSCache) Server() error {
	var err error
	cache.server.PacketConn, err = net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 0,
	})
	if err != nil {
		return err
	}

	cache.server.Handler = &handler{cache: cache}

	serverStarted := make(chan error, 2)

	cache.server.NotifyStartedFunc = func() {
		serverStarted <- nil
	}

	cache.closed = false

	go func() {
		defer cache.server.PacketConn.Close()
		if err := cache.server.ActivateAndServe(); err != nil {
			if !cache.closed {
				cache.Logger.Debug("DNSCache got error", zap.String("error", err.Error()))
				serverStarted <- err
			}
		}
		cache.Logger.Debug("DNS Server exited")
	}()

	if err := <-serverStarted; err != nil {
		return err
	}

	var ok bool
	cache.localAddr, ok = cache.server.PacketConn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return errors.New("LocalAddr is not an UDPAddr")
	}
	cache.Logger.Debug("DNSCache is running", zap.String("address", cache.localAddr.String()))

	return nil
}

func (cache *DNSCache) getCacheEntries(name string, Qclass uint16, Qtype uint16) ([]dns.RR, error) {
	now := time.Now()
	var entries []dns.RR
again:
	for i := 0; i < len(cache.entries); i++ {
		hdr := cache.entries[i].RR.Header()
		if hdr.Class == Qclass && hdr.Rrtype == Qtype && hdr.Name == name {
			if !cache.entries[i].ValidUntil.IsZero() && cache.entries[i].ValidUntil.Before(now) {
				// dns entry is old
				if err := cache.refreshCacheEntries(&cache.entries[i]); err != nil {
					return nil, err
				}
				entries = []dns.RR{}
				goto again
			}
			entries = append(entries, cache.entries[i].RR)
		}
	}
	return entries, nil
}

func (cache *DNSCache) refreshCacheEntries(entry *cacheEntry) error {
	// we are allready refreshing this item
	if entry.Refreshing {
		return nil
	}
	server := entry.OriginServer
	net := entry.OriginServerNet
	s := entry.RR.Header().String()
	cache.Logger.Debug("DNSCache refreshing", zap.String("host", entry.RR.Header().Name))
	cache.mu.Lock()
	entry.Refreshing = true
	for i := len(cache.entries) - 1; i >= 0; i-- {
		if len(cache.entries[i].OriginServer) > 0 && cache.entries[i].RR.Header().String() == s {
			cache.entries = append(cache.entries[:i], cache.entries[i+1:]...)
		}
	}
	cache.mu.Unlock()
	return cache.ResolveAndAdd(server, net, entry.RR.Header().Name, entry.RR.Header().Class, entry.RR.Header().Rrtype)
}

func (cache *DNSCache) addCacheEntries(name string, entries []dns.RR, dst *[]dns.RR) {
	for i := 0; i < len(entries); i++ {
		entries[i].Header().Name = name
		*dst = append(*dst, entries[i])
	}
}

// Close closes an DNSCache Server (started with Server())
func (cache *DNSCache) Close() {
	cache.closed = true
	cache.server.PacketConn.Close()
}

// Resolver returns an net.Resolver that can be used
func (cache *DNSCache) Resolver() *net.Resolver {
	if cache.closed {
		return nil
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return net.DialUDP("udp", nil, cache.localAddr)
		},
	}
}

// ResolveAndAdd resolves and adds the result to the cache
func (cache *DNSCache) ResolveAndAdd(dnsServer, net string, host string, Qclass uint16, Qtype uint16) error {
	var client dns.Client
	var msg dns.Msg

	client.Net = net
	client.DialTimeout = cache.DialTimeout
	sanitizeHost(&host)

	msg.Question = []dns.Question{
		dns.Question{
			Name:   host,
			Qclass: Qclass,
			Qtype:  Qtype,
		},
	}
	msg.RecursionDesired = true
	var r *dns.Msg
	var err error
	for i := 0; i < cache.MaxLookupAttempts; i++ {
		cache.Logger.Debug("Resolving DNS", zap.String("server", dnsServer), zap.String("host", host), zap.Uint16("class", Qclass), zap.Uint16("type", Qtype))
		r, _, err = client.Exchange(&msg, dnsServer)
		if err != nil || r.Rcode != dns.RcodeSuccess {
			continue
		}
		break
	}

	if err != nil {
		return fmt.Errorf("Unable to lookup host `%s': %s", host, err.Error())
	}
	if r.Rcode != dns.RcodeSuccess {
		return fmt.Errorf("Unable to lookup host `%s': Response code was %d, expected %d", host, r.Rcode, dns.RcodeSuccess)
	}

	if size := len(r.Answer); size > 0 {
		entries := make([]cacheEntry, size)

		for i, e := range r.Answer {
			entries[i].OriginServer = dnsServer
			entries[i].OriginServerNet = net
			entries[i].ValidUntil = time.Now().Add(time.Duration(e.Header().Ttl) * time.Second)
			entries[i].RR = e
		}

		cache.Logger.Debug("Adding resolved entries to cache", zap.String("server", dnsServer), zap.String("host", host), zap.Any("entries", entries))

		cache.mu.Lock()
		cache.entries = append(cache.entries, entries...)
		cache.mu.Unlock()
	}
	return nil
}

// Add adds a static entry to the cache
func (cache *DNSCache) Add(record ...dns.RR) error {
	cache.Logger.Debug("Adding entries to cache", zap.Any("entries", record))

	entries := make([]cacheEntry, len(record))

	for i, e := range record {
		entries[i].RR = e
	}

	cache.mu.Lock()
	cache.entries = append(cache.entries, entries...)
	cache.mu.Unlock()
	return nil
}

// Truncate truncates the cache
func (cache *DNSCache) Truncate() error {
	cache.Logger.Debug("Truncating cache")
	cache.mu.Lock()
	cache.entries = []cacheEntry{}
	cache.mu.Unlock()
	return nil
}

// Addr returns the listening address of the server
func (cache *DNSCache) Addr() string {
	if cache.closed {
		return ""
	}
	return cache.server.PacketConn.LocalAddr().String()
}

// Lookup looks up an entry in the cache
func (cache *DNSCache) Lookup(host string, Qclass uint16, Qtype uint16) ([]dns.RR, error) {
	sanitizeHost(&host)
	return cache.getCacheEntries(host, Qclass, Qtype)
}

func sanitizeHost(host *string) {
	*host = strings.TrimRightFunc(strings.ToLower(*host), func(r rune) bool {
		return r == '.'
	}) + "."
}
