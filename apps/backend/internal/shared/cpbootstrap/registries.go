package cpbootstrap

import (
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

type rpcRoutes struct {
	owners map[string]string
	paths  []string
	byPath map[string]http.Handler

	errorNet connect.Interceptor

	drain connect.Interceptor
}

func newRPCRoutes(errorNet, drain connect.Interceptor) *rpcRoutes {
	return &rpcRoutes{
		owners:   map[string]string{},
		paths:    nil,
		byPath:   map[string]http.Handler{},
		errorNet: errorNet,
		drain:    drain,
	}
}

func (r *rpcRoutes) forModule(module string) RPCRegistrar {
	return moduleRoutes{module: module, routes: r}
}

func (r *rpcRoutes) mountOn(router *http.ServeMux, middlewares func(http.Handler) http.Handler) {
	for _, path := range r.paths {
		router.Handle(path, middlewares(r.byPath[path]))
	}
}

type moduleRoutes struct {
	module string
	routes *rpcRoutes
}

func (m moduleRoutes) Mount(build ServiceBuilder, interceptors ...connect.Interceptor) error {
	if build == nil {
		return fmt.Errorf("module %s mounted a nil service", m.module)
	}

	chain := append([]connect.Interceptor{m.routes.errorNet, m.routes.drain}, interceptors...)

	path, handler := build(connect.WithInterceptors(chain...))

	if path == "" {
		return fmt.Errorf("module %s mounted a handler on no path", m.module)
	}
	if handler == nil {
		return fmt.Errorf("module %s mounted a nil handler on %q", m.module, path)
	}
	if owner, taken := m.routes.owners[path]; taken {
		return fmt.Errorf(
			"module %s claims %q, which %s already serves: the router would panic on the second one",
			m.module, path, owner,
		)
	}

	m.routes.owners[path] = m.module
	m.routes.paths = append(m.routes.paths, path)
	m.routes.byPath[path] = handler

	return nil
}

type runnerRegistry struct {
	runners []Runner
}

func newRunnerRegistry() *runnerRegistry {
	return &runnerRegistry{runners: nil}
}

func (r *runnerRegistry) Add(runner Runner) {
	r.runners = append(r.runners, runner)
}

func (r *runnerRegistry) count() int {
	return len(r.runners)
}

type namedCloser struct {
	name  string
	close func() error
}

type closerRegistry struct {
	closers []namedCloser
}

func newCloserRegistry() *closerRegistry {
	return &closerRegistry{closers: nil}
}

func (c *closerRegistry) Add(name string, close func() error) {
	c.closers = append(c.closers, namedCloser{name: name, close: close})
}

func (c *closerRegistry) all() []namedCloser {
	reversed := make([]namedCloser, 0, len(c.closers))
	for i := len(c.closers) - 1; i >= 0; i-- {
		reversed = append(reversed, c.closers[i])
	}

	return reversed
}
