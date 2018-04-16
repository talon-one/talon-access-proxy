package talon_access_proxy

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/miekg/dns"
	"github.com/talon-one/talon-access-proxy/dnscache"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate go run generate.go

// VersionHash represents the git sha1 which this version was built on
var VersionHash = "Unknown/CustomBuild"

// Version represents the build version of talon-access-proxy
var Version = "Unknown/CustomBuild"

// BuildDate represents the build date of talon-access-proxy
var BuildDate = "Unknown/CustomBuild"

// Tap implements the talon-access-proxy functionality
type Tap struct {
	Config   Config
	mux      *mux
	dnscache *dnscache.DNSCache
	client   http.Client

	logger *zap.Logger
}

// New creates a new instance of the Tap type
func New(config Config) (*Tap, error) {
	// make sure the config is valid
	if err := config.SetDefaults(); err != nil {
		return nil, err
	}
	// create a tap instance
	t := &Tap{
		Config: config,
		logger: config.Logger.With(zap.String("tag", "Tap")),
	}
	// create an http mux instance that handles incoming requests
	t.mux = newMux(t)

	t.dnscache = dnscache.New(config.Logger.With(zap.String("tag", "DNSCache")))

	// make sure talonHost has no port in it
	talonHost := t.Config.talonAPIUrl.Hostname()

	// if the talon service is a hostname, resolve it
	if govalidator.IsDNSName(talonHost) {
		err := t.dnscache.ResolveAndAdd(t.Config.DNSServer, "udp", talonHost, dns.ClassINET, dns.TypeA)
		if err != nil {
			return nil, err
		}
		err = t.dnscache.ResolveAndAdd(t.Config.DNSServer, "udp", talonHost, dns.ClassINET, dns.TypeAAAA)
		if err != nil {
			return nil, err
		}
	} else if !govalidator.IsIP(talonHost) {
		return nil, errors.New("TalonURL does not contain a valid host part")
	}

	if err := t.dnscache.Server(); err != nil {
		return nil, err
	}

	// create http client
	t.client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  t.dnscache.Resolver(),
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          t.Config.MaxConnections,
		IdleConnTimeout:       0,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return t, nil
}

// Handler returns an http.Handler instance
func (t *Tap) Handler() http.Handler {
	return t.mux
}

// Close the tap instance
func (t *Tap) Close() {
}

func (t *Tap) doHTTPRequest(r *http.Request) (*http.Response, error) {
	r.URL.Host = t.Config.talonAPIUrl.Host
	r.URL.Scheme = t.Config.talonAPIUrl.Scheme
	req := http.Request{
		Method:        r.Method,
		URL:           r.URL,
		Header:        r.Header,
		Body:          r.Body,
		ContentLength: r.ContentLength,
		Close:         false,
	}

	logger := t.logger

	if t.logger.Core().Enabled(zap.DebugLevel) {
		logger = logger.With(zap.Uint32("id", crc32.ChecksumIEEE([]byte(fmt.Sprintf("%s-%s-%d-%d", req.Method, req.URL.String(), req.ContentLength, time.Now().Unix())))))
		logger.Debug("Performing Request", zap.String("method", req.Method), zap.String("url", req.URL.String()), zap.Int64("content-length", req.ContentLength), zap.Any("header", req.Header))
	}

	if len(t.Config.Application) > 0 {
		if err := t.applicationSpecificHeaders(logger, r, &req); err != nil {
			logger.Debug("Request got error", zap.String("error", err.Error()))
			return nil, err
		}
	}

	res, err := t.client.Do(&req)
	if err != nil {
		logger.Debug("Request got error", zap.String("error", err.Error()))
	} else {
		logger.Debug("Request succeeded",
			zap.Int("statusCode", res.StatusCode),
			zap.Int64("content-length", res.ContentLength),
			zap.Any("header", res.Header),
		)
	}

	return res, err
}

func extractApplicationID(r *http.Request) string {
	if header := r.Header.Get("Api-Key"); len(header) > 0 {
		fields := strings.Split(header, ".")
		for _, field := range fields {
			tokens := strings.Split(field, "=")
			if len(tokens) < 2 {
				return ""
			}
			if strings.TrimSpace(tokens[0]) == "application" {
				return strings.TrimSpace(tokens[1])
			}
		}
	}

	if header := r.Header.Get("Content-Signature"); len(header) > 0 {
		fields := strings.Split(header, ";")
		for _, field := range fields {
			tokens := strings.Split(field, "=")
			if len(tokens) < 2 {
				return ""
			}
			if strings.TrimSpace(tokens[0]) == "signer" {
				return strings.TrimSpace(tokens[1])
			}
		}
	}
	return ""
}

func (t *Tap) applicationSpecificHeaders(logger *zap.Logger, incomingRequest, outgoingRequest *http.Request) error {
	appID := extractApplicationID(incomingRequest)
	if len(appID) <= 0 {
		return nil
	}
	for id, config := range t.Config.Application {
		if !strings.EqualFold(id, appID) {
			continue
		}
		if config.CalculateHMAC && incomingRequest.Body != nil && strings.EqualFold(incomingRequest.Header.Get("Content-Type"), "application/json") {
			logger.Debug("Calculating HMAC")
			mac := hmac.New(md5.New, config.applicationKeyBytes)
			// copy the body
			var buffer bytes.Buffer
			w := io.MultiWriter(&buffer, mac)
			if _, err := io.Copy(w, incomingRequest.Body); err != nil {
				return err
			}

			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("Copied body", zap.ByteString("body", buffer.Bytes()))
			}

			signature := hex.EncodeToString(mac.Sum(nil))
			outgoingRequest.Header.Set("Content-Signature", fmt.Sprintf("signer=%s;signature=%s", id, signature))
			logger.Debug("HMAC Calculated", zap.String("signer", id), zap.String("signature", signature))
			outgoingRequest.Body = ioutil.NopCloser(&buffer)
		}
		if len(config.ApplicationToken) > 0 {
			logger.Debug("Adding Api-Key to request")
			outgoingRequest.Header.Set("Api-Key", fmt.Sprintf("application=%s.token=%s", id, config.ApplicationToken))
		}
		return nil
	}
	return nil
}
