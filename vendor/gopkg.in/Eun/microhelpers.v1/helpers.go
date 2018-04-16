package microhelpers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

func ListenAndServe(addresses []string, handler http.Handler, logger ...io.Writer) error {
	size := len(addresses)
	if size <= 0 {
		return errors.New("No addresses to listen on")
	}

	var firstLogger io.Writer

	for i := 0; i < len(logger); i++ {
		if logger[i] != nil {
			firstLogger = logger[i]
			break
		}
	}

	errChan := make(chan error, size)

	for i := 0; i < size; i++ {
		go func(address string) {
			if firstLogger != nil {
				fmt.Fprintf(firstLogger, "Listening on %s\n", address)
			}
			if err := http.ListenAndServe(address, handler); err != nil {
				errChan <- err
			}
		}(addresses[i])
	}

	return <-errChan
}
