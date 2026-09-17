package cpbootstrap

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"
)

// internalDialer is the one place that knows the internal transport is loopback HTTP.
type internalDialer struct {
	address string
}

var internalClient = &http.Client{Transport: http.DefaultTransport}

func (d internalDialer) Dial() (connect.HTTPClient, string, error) {
	if d.address == "" {
		return nil, "", errors.New("httpServer.internalBindAddress is empty: no module can be called")
	}

	return internalClient, "http://" + d.address, nil
}
