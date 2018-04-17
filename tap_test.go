package talon_access_proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func TestAPICallWithHMAC(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	// from the https://developers.talon.one/docs/
	tap, err := New(Config{
		TalonAPI: "https://demo.talon.one",
		Application: map[string]*ApplicationConfig{
			"73": &ApplicationConfig{
				CalculateHMAC:  true,
				ApplicationKey: "e3b620ed8144f292",
			},
		},
		Logger: logger,
	})
	require.NoError(t, err)

	server := httptest.NewServer(tap.Handler())

	defer server.Close()

	var buffer bytes.Buffer
	require.NoError(t, json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"attributes": map[string]interface{}{
			"Email": "carltonb@hushmail.com",
		},
	}))

	req, err := http.NewRequest("PUT", server.URL+"/v1/customer_profiles/165f239c", &buffer)
	require.NoError(t, err)
	req.Header.Set("Content-Signature", "signer=73")
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, Version, res.Header.Get("X-Tap"))
}

func BenchmarkTap(b *testing.B) {
	server := httptest.NewServer(nil)
	defer server.Close()

	logger, err := zap.NewProduction()
	require.NoError(b, err)
	tap, err := New(Config{
		TalonAPI: server.URL,
		Logger:   logger,
	})
	require.NoError(b, err)
	defer tap.Close()

	tapServer := httptest.NewServer(tap.Handler())
	defer tapServer.Close()

	c := &fasthttp.HostClient{
		Addr: tapServer.Listener.Addr().String(),
	}
	for i := 0; i < b.N; i++ {
		statusCode, _, err := c.Get(nil, tapServer.URL)
		require.NoError(b, err)
		require.Equal(b, http.StatusNotFound, statusCode)
	}
}
