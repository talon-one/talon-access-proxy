package talon_access_proxy

import (
	"io"
	"net/http"

	"go.uber.org/zap"
)

type mux struct {
	Tap    *Tap
	Logger *zap.Logger
}

func newMux(t *Tap) *mux {
	return &mux{
		Tap:    t,
		Logger: t.Config.Logger.With(zap.String("tag", "Mux")),
	}
}

func (mux *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.Logger.Debug("Got Request", zap.String("method", r.Method), zap.String("url", r.URL.String()), zap.Int64("content-length", r.ContentLength), zap.Any("headers", r.Header))

	response, err := mux.Tap.doHTTPRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// copy all headers
	for header, value := range response.Header {
		w.Header().Set(header, value[0])
	}
	w.Header().Set("X-TAP", Version)

	mux.Logger.Debug("Sending Response",
		zap.Int64("content-length", response.ContentLength),
		zap.Any("headers", w.Header()))

	w.WriteHeader(response.StatusCode)
	if response.Body != nil {
		_, err := io.Copy(w, response.Body)
		if err != nil && err != io.EOF {
			mux.Logger.Error("Unable to copy body", zap.Error(err))
		}
	}
}
