package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	hjson "github.com/Eun/hjson-go"
	"github.com/Eun/microhelpers"
	"github.com/mitchellh/mapstructure"
	tap "github.com/talon-one/talon-access-proxy"
	"go.uber.org/zap"
)

// Config represents a config for the talon-access-proxy application
type Config struct {
	Address        string
	Root           string
	MaxConnections *int
	tap.Config     `mapstructure:",squash"`
}

func readConfigs() ([]Config, error) {
	// determinate which config file we should use
	configFile, err := microhelpers.ParseString([]string{"config", "c"}, []string{"APP_CONFIG"}, "config.json", os.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("Unable to read config: %s", err.Error())
	}

	if len(configFile) <= 0 {
		return nil, fmt.Errorf("Invalid config file specified, use --config parameter or the APP_CONFIG environment variable")
	}

	// try to read config
	buffer, err := ioutil.ReadFile(configFile)
	if err != nil {
		if !strings.ContainsRune(strings.Replace(filepath.ToSlash(configFile), "./", "", -1), '/') {
			dir, err := os.Getwd()
			if err == nil {
				return nil, fmt.Errorf("Unable to read config file `%s' in %s", configFile, dir)
			}
		}
		return nil, fmt.Errorf("Unable to read config file `%s'", configFile)
	}

	var data interface{}
	if err := hjson.Unmarshal(buffer, &data); err != nil {
		return nil, fmt.Errorf("Unable to read config file `%s': %s", configFile, err.Error())
	}

	switch v := data.(type) {
	case map[string]interface{}:
		config, err := readConfig(v)
		if err != nil {
			return nil, err
		}
		return []Config{config}, nil
	case []interface{}:
		var configs []Config
		for i := 0; i < len(v); i++ {
			if data, ok := v[i].(map[string]interface{}); ok {
				config, err := readConfig(data)
				if err != nil {
					return nil, err
				}
				configs = append(configs, config)
			} else {
				return nil, fmt.Errorf("Unknown config type: %T", v[i])
			}
		}
		return configs, nil
	default:
		return nil, fmt.Errorf("Unknown config type: %T", data)
	}
}

func readConfig(dat map[string]interface{}) (config Config, err error) {
	if err := mapstructure.Decode(dat, &config); err != nil {
		return config, fmt.Errorf("Unable to read config file `%s': %s", configFile, err.Error())
	}

	port, err := microhelpers.ParseInt([]string{"port", "p"}, []string{"PORT", "APP_PORT", "HTTP_PLATFORM_PORT", "ASPNETCORE_PORT"}, 0, os.Args)
	if err != nil {
		return config, fmt.Errorf("Unable to read port: %s", err.Error())
	}
	config.Address, err = microhelpers.ParseString([]string{"address", "a"}, []string{"ADDRESS", "APP_ADDRESS"}, config.Address, os.Args)
	if err != nil {
		return config, fmt.Errorf("Unable to read address: %s", err.Error())
	}
	if len(config.Address) <= 0 {
		config.Address = fmt.Sprintf(":%d", port)
	} else {
		if _, _, err := net.SplitHostPort(config.Address); err != nil {
			return config, fmt.Errorf("Unable to find port in address")
		}
	}
	config.Root, err = microhelpers.ParseString([]string{"root", "r"}, []string{"APP_ROOT"}, config.Root, os.Args)
	if err != nil {
		return config, fmt.Errorf("Unable to read root: %s", err.Error())
	}
	config.Root = "/" + strings.Trim(filepath.ToSlash(config.Root), "/")

	config.TalonAPI, err = microhelpers.ParseString([]string{"talon", "t"}, nil, config.TalonAPI, os.Args)
	if err != nil {
		return config, fmt.Errorf("Unable to read talon: %s", err.Error())
	}

	debug, err := microhelpers.ParseInt([]string{"debug"}, []string{"DEBUG"}, 0, os.Args)
	if err != nil {
		return config, fmt.Errorf("Unable to read debug: %s", err.Error())
	}

	if debug <= 0 {
		config.Config.Logger, err = zap.NewProduction()
	} else {
		config.Config.Logger, err = zap.NewDevelopment()
		config.Config.Logger.Debug("Debug is enabled")
	}
	if err != nil {
		return config, fmt.Errorf("Unable to create logger: %s", err.Error())
	}

	config.Logger = config.Logger.With(zap.String("address", config.Address), zap.String("api", config.TalonAPI))

	if config.MaxConnections == nil {
		config.Config.MaxConnections = 100
	} else {
		config.Config.MaxConnections = *config.MaxConnections
	}
	return config, nil
}
