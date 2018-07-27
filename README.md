# talon-access-proxy
This is a standalone http proxy that caches connections to a server.

It is used for [talon.one](https://talon.one) as a middleware to reduce latency to the talon.one api endpoint.


# Installation
You find releases in [Github Releases](https://github.com/talon-one/talon-access-proxy/releases) section.

Or you can use `go install`:
```bash
go install github.com/talon-one/talon-access-proxy/cmd/talon-access-proxy
```

# Usage
```
# talon-access-proxy --help

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
[
    {
        // Address to listen on
        "Address": "127.0.0.1:8000"

        // Root path
        "Root": "/"

        // Talon api
        "TalonAPI": "https://demo.talon.one"

        // DNS Server that should be used for lookups
        "DNSServer": "8.8.8.8:53"

        // How many concurrent connections should be used
        "MaxConnections": 100
        

        // Application specific settings
        Application: {
            // Application with the ID 1
            "1": {
                // Calculate HMAC for each request
                "CalculateHMAC": false

                // Application Key (required for CalculateHMAC)
                ApplicationKey: "deadbeef"
            }
        }
    },
    {
        // Open a second instance
        "Address": "127.0.0.1:8001"
        "TalonAPI": "https://demo.talon.one"
    },
]
```


