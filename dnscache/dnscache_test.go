package dnscache

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCache(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cache := New(logger)
	defer cache.Close()

	cache.Add(&dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "example.com.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Txt: []string{"Hello World"},
	})

	require.NoError(t, cache.Server())

	entries, err := cache.Resolver().LookupTXT(context.Background(), "example.com")
	require.NoError(t, err)
	require.EqualValues(t, []string{"Hello World"}, entries)
}

func TestCacheMiss(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cache := New(logger)
	defer cache.Close()

	require.NoError(t, cache.Server())

	_, err = cache.Resolver().LookupTXT(context.Background(), "example.com")
	require.Error(t, err)
}

func TestCacheRefresh(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	cache := New(logger)
	defer cache.Close()

	cache2 := New(logger.With(zap.Bool("sub", true)))
	defer cache2.Close()
	cache2.Add(&dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "example.com.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Txt: []string{"Hello World"},
	})
	require.NoError(t, cache2.Server())

	require.NoError(t, cache.ResolveAndAdd(cache2.Addr(), "udp", "example.com", dns.ClassINET, dns.TypeTXT))

	cache.entries[0].ValidUntil = time.Now().Add(time.Hour * -1)

	require.NoError(t, cache.Server())

	entries, err := cache.Resolver().LookupTXT(context.Background(), "example.com")
	require.NoError(t, err)
	require.EqualValues(t, []string{"Hello World"}, entries)
}
