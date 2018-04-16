package talon_access_proxy

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"

	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

// Config contains settings for the proxy
type Config struct {
	// TalonAPI is the URL to use
	TalonAPI    string
	talonAPIUrl url.URL
	// DNSServer to use for dns lookups (Default is 8.8.8.8:53)
	DNSServer string
	// MaxConnections to use
	MaxConnections int

	// Application ID
	Application map[string]*ApplicationConfig

	// Logger to write data to
	Logger *zap.Logger
}

type ApplicationConfig struct {
	// Calculate HMAC
	CalculateHMAC bool
	// Application Key
	ApplicationKey      string
	applicationKeyBytes []byte
	// ApplicationToken to use
	ApplicationToken string
}

// SetDefaults validates and sets defaults for Config
func (config *Config) SetDefaults() error {
	u, err := url.Parse(config.TalonAPI)
	if err != nil {
		return fmt.Errorf("Unable to parse TalonAPI: %s", err.Error())
	}
	config.talonAPIUrl = *u

	if len(config.talonAPIUrl.Scheme) <= 0 {
		config.talonAPIUrl.Scheme = "https"
	}

	if len(config.DNSServer) <= 0 {
		config.DNSServer = "8.8.8.8:53"
	}

	if config.MaxConnections < 0 {
		config.MaxConnections = 0
	}

	for id, key := range config.Application {
		var err error
		config.Application[id].applicationKeyBytes, err = hex.DecodeString(key.ApplicationKey)
		if err != nil {
			return fmt.Errorf("ApplicationKey is invalid, (ApplicationID=%s)", id)
		}
	}

	if err := config.createLogger(); err != nil {
		return err
	}

	return config.testConfig()
}

func (config *Config) testConfig() error {
	if len(config.TalonAPI) <= 0 {
		return errors.New("TalonAPI is not set")
	}
	if !govalidator.IsDialString(config.DNSServer) {
		return errors.New("DNSServer is invalid, must be in the form of host:port")
	}
	for _, key := range config.Application {
		if key.CalculateHMAC {
			if len(key.applicationKeyBytes) <= 0 {
				return errors.New("ApplicationKey must be set if you want to use the CalculateHMAC function")
			}
		}
	}

	return nil
}

func (config *Config) createLogger() error {
	// create a logger if there is none set
	if config.Logger == nil {
		var err error
		config.Logger, err = zap.NewProduction()
		if err != nil {
			return err
		}
	}
	return nil
}
