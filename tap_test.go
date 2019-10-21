package talon_access_proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

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
