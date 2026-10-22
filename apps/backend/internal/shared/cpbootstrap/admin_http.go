package cpbootstrap

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"connectrpc.com/connect"
)

type adminHandler struct {
	module  string
	pattern string
	handler http.Handler
}

type adminHandlers struct {
	handlers []adminHandler
}

func (a *adminHandlers) forModule(module string) AdminHTTPRegistrar {
	return moduleAdminHandlers{module: module, handlers: a}
}

// Outside the gate's middlewares: what is served here says who may call itself, and it is not the admin page.
func (a *adminHandlers) mountOn(router *http.ServeMux, middlewares func(http.Handler) http.Handler) error {
	for _, mounted := range a.handlers {
		if err := handle(router, mounted.pattern, middlewares(mounted.handler)); err != nil {
			return fmt.Errorf("module %s: %w", mounted.module, err)
		}
	}

	return nil
}

// The router panics on a pattern that conflicts with one it holds, an admin service's included.
func handle(router *http.ServeMux, pattern string, handler http.Handler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("cannot serve %q on the admin listener: %v", pattern, recovered)
		}
	}()

	router.Handle(pattern, handler)
	return nil
}

type moduleAdminHandlers struct {
	module   string
	handlers *adminHandlers
}

func (m moduleAdminHandlers) Handle(pattern string, handler http.Handler) error {
	if pattern == "" || handler == nil {
		return fmt.Errorf("module %s handed the admin listener an empty pattern or a nil handler", m.module)
	}
	for _, mounted := range m.handlers.handlers {
		if mounted.pattern == pattern {
			return fmt.Errorf("module %s claims %q on the admin listener, which %s already serves", m.module, pattern, mounted.module)
		}
	}

	m.handlers.handlers = append(m.handlers.handlers, adminHandler{module: m.module, pattern: pattern, handler: handler})
	return nil
}

type adminDialer struct {
	address string
}

// An address with no host, or the unspecified one, listens on loopback too, and loopback is what this process dials.
func (d adminDialer) Dial() (connect.HTTPClient, string, error) {
	if d.address == "" {
		return nil, "", errors.New("httpServer.adminBindAddress is empty: there is no admin listener to call")
	}

	host, port, err := net.SplitHostPort(d.address)
	if err != nil {
		return nil, "", fmt.Errorf("httpServer.adminBindAddress %q is not a host:port: %w", d.address, err)
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}

	return internalClient, "http://" + net.JoinHostPort(host, port), nil
}
