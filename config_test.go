package talon_access_proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigSetDefaults(t *testing.T) {
	t.Run("Minimal config", func(t *testing.T) {
		config := &Config{
			TalonAPI: "https://demo.talon.one",
		}
		require.NoError(t, config.SetDefaults())
	})
	t.Run("Invalid config", func(t *testing.T) {
		config := &Config{}
		require.Error(t, config.SetDefaults())
	})
	t.Run("Invalid DNS", func(t *testing.T) {
		config := &Config{
			TalonAPI:  "https://demo.talon.one",
			DNSServer: "1.2.3.4",
		}
		require.Error(t, config.SetDefaults())
	})
	t.Run("Invalid MaxConnections", func(t *testing.T) {
		config := &Config{
			TalonAPI:       "https://demo.talon.one",
			MaxConnections: -1,
		}
		require.NoError(t, config.SetDefaults())
		require.Equal(t, 0, config.MaxConnections)
	})
	t.Run("Invalid ApplicationKey", func(t *testing.T) {
		config := &Config{
			TalonAPI: "https://demo.talon.one",
			Application: map[string]*ApplicationConfig{
				"1": &ApplicationConfig{
					ApplicationKey: "Hello",
				},
			},
		}
		require.Error(t, config.SetDefaults())
	})
	t.Run("CalculateHMAC needs ApplicationKey", func(t *testing.T) {
		config := &Config{
			TalonAPI: "https://demo.talon.one",
			Application: map[string]*ApplicationConfig{
				"1": &ApplicationConfig{
					CalculateHMAC: true,
				},
			},
		}
		require.Error(t, config.SetDefaults())
	})
	t.Run("No Scheme", func(t *testing.T) {
		config := &Config{
			TalonAPI: "demo.talon.one",
		}
		require.NoError(t, config.SetDefaults())
		require.Equal(t, "https://demo.talon.one", config.talonAPIUrl.String())
	})
}
