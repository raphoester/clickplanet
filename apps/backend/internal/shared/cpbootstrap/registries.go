package cpbootstrap

import (
	"context"
	"fmt"
	"net/http"
)

// rpcRoutes collects what every module mounted, so the router is built once
// from the whole set rather than edited by each module in turn.
type rpcRoutes struct {
	owners map[string]string
	paths  []string
	byPath map[string]http.Handler
}

func newRPCRoutes() *rpcRoutes {
	return &rpcRoutes{
		owners: map[string]string{},
		paths:  nil,
		byPath: map[string]http.Handler{},
	}
}

func (r *rpcRoutes) forModule(module string) RPCRegistrar {
	return moduleRoutes{module: module, routes: r}
}

// mountOn wraps every route in the same middleware stack. A module cannot add
// its own here: a policy that has to reach one service is an interceptor, which
// is the only layer that can tell one procedure from another.
func (r *rpcRoutes) mountOn(router *http.ServeMux, middlewares func(http.Handler) http.Handler) {
	for _, path := range r.paths {
		router.Handle(path, middlewares(r.byPath[path]))
	}
}

type moduleRoutes struct {
	module string
	routes *rpcRoutes
}

func (m moduleRoutes) Mount(path string, handler http.Handler) error {
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

type namedRunner struct {
	name string
	run  func(ctx context.Context)
}

type runnerRegistry struct {
	runners []namedRunner
}

func newRunnerRegistry() *runnerRegistry {
	return &runnerRegistry{runners: nil}
}

func (r *runnerRegistry) Add(name string, run func(ctx context.Context)) {
	r.runners = append(r.runners, namedRunner{name: name, run: run})
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

// all returns the cleanups in reverse registration order, which is dependency
// order reversed: a module is built after the ones before it, so it is closed
// before them.
func (c *closerRegistry) all() []namedCloser {
	reversed := make([]namedCloser, 0, len(c.closers))
	for i := len(c.closers) - 1; i >= 0; i-- {
		reversed = append(reversed, c.closers[i])
	}

	return reversed
}
