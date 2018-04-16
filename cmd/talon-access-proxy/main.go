package main

//go:generate go run generate.go

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"os/signal"

	"github.com/araddon/dateparse"
	tap "github.com/talon-one/talon-access-proxy"
	"go.uber.org/zap"
	"gopkg.in/Eun/microhelpers.v1"
)

var configFile = ""

const releasesURL = "https://api.github.com/repos/talon-one/talon-access-proxy/releases"

func main() {
	showHelp, err := microhelpers.ParseBool([]string{"help", "h"}, nil, false, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read help: %s\n", err.Error())
		os.Exit(1)
	}

	if showHelp {
		fmt.Printf(strings.TrimSpace(`
talon-access-proxy is a proxy for the talon service api

Usage:

    talon-access-proxy [option]

The options are:

    -h, --help         show this help
    -c, --config       specify the config file to use
    -p, --port         specify a port to listen on
    -a, --address      listen on this address (host:port), overrides --port
    -r, --root=/       specify a root path for this service
    -t, --talon=       specify the talon api url to use
    -v, --version      show the version

Environment settings:

You can set various environment variables in conjunction with the options, note that
options overwrite the corresponding environment variable.

    APP_CONFIG         specify the config file to use
    PORT               specify a port to listen on
    APP_PORT
    HTTP_PLATFORM_PORT
    ASPNETCORE_PORT
    ADDRESS            listen on this address (host:port), overrides PORT
    APP_ADDRESS
    APP_ROOT           specify a root path for this service

The config

The config specified with --config or APP_CONFIG can also be used to specify options

Sample Config:
%s
`), configFile)
		os.Exit(0)
	}

	version, err := microhelpers.ParseBool([]string{"version", "v"}, nil, false, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read version: %s\n", err.Error())
		os.Exit(1)
	}
	if version {
		fmt.Printf("talon-access-proxy %s %s %s\n", tap.Version, tap.VersionHash, tap.BuildDate)
		return
	}

	// read config, and create a tap Config
	configs, err := readConfigs()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	go checkUpdates()

	errChan := make(chan error)

	for i := 0; i < len(configs); i++ {
		defer configs[i].Logger.Sync()
		go runConfig(configs[i], errChan)
	}

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	select {
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
	case <-signalChan:
		return
	}
}

func runConfig(config Config, errChan chan<- error) {
	tap, err := tap.New(config.Config)
	if err != nil {
		errChan <- fmt.Errorf("Unable to create tap: %s", err.Error())
		return
	}

	config.Logger.Debug("Config", zap.String("talon", config.TalonAPI))

	handler := tap.Handler()

	if config.Root != "/" {
		config.Logger.Debug("Root is set", zap.String("root", config.Root))
		mux := http.NewServeMux()
		mux.Handle(config.Root, http.RedirectHandler(config.Root+"/", http.StatusTemporaryRedirect))
		mux.Handle(config.Root+"/", http.StripPrefix(config.Root, handler))
		handler = mux
	}

	config.Logger.Info("Listening")
	if err := http.ListenAndServe(config.Address, handler); err != nil {
		errChan <- fmt.Errorf("Listen Error: %s", err.Error())
	}
}

func checkUpdates() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	logger = logger.With(zap.String("tag", "Update"))
	resp, err := http.DefaultClient.Get(releasesURL)
	if err != nil {
		logger.Error("Fetching update information failed", zap.String("error", err.Error()))
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.Debug("Invalid status", zap.Int("status", resp.StatusCode))
		return
	}

	var data []map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.Error("Invalid update format", zap.String("error", err.Error()))
		return
	}

	getStringField := func(m map[string]interface{}, key string) *string {
		var k string
		var v interface{}

		for k, v = range m {
			if strings.EqualFold(key, k) {
				break
			}
		}
		if v == nil {
			return nil
		}
		if s, ok := v.(string); ok {
			return &s
		}
		return nil
	}

	hasSliceField := func(m map[string]interface{}, key string) bool {
		for k, v := range m {
			if strings.EqualFold(key, k) {
				if s, ok := v.([]interface{}); ok {
					if len(s) > 0 {
						return true
					}
				}
				return false
			}
		}
		return false
	}

	var lastUpdate struct {
		time    time.Time
		data    *map[string]interface{}
		version string
	}

	for i := 0; i < len(data); i++ {
		publishedAt := getStringField(data[i], "published_at")
		if publishedAt == nil {
			logger.Error("published_at is not a present or a string")
			continue
		}
		tagName := getStringField(data[i], "tag_name")
		if tagName == nil {
			logger.Error("tag_name is not a present or a string")
			continue
		}

		if !hasSliceField(data[i], "assets") {
			logger.Debug("assets is not a present or a slice")
			continue
		}

		t, err := dateparse.ParseAny(*publishedAt)
		if err != nil {
			logger.Error("Invalid time format", zap.String("time", *publishedAt), zap.String("error", err.Error()))
			continue
		}

		if lastUpdate.time.Before(t) {
			lastUpdate.time = t
			lastUpdate.data = &data[i]
			lastUpdate.version = *tagName
		}
	}

	if lastUpdate.data == nil {
		logger.Debug("Got latest version")
		return
	}

	if strings.Compare(lastUpdate.version, tap.Version) <= 0 {
		logger.Debug("Got latest version")
		return
	}

	logger.Info(fmt.Sprintf("There is a new version available: %s", lastUpdate.version))
}
