package talon_access_proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
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
}
