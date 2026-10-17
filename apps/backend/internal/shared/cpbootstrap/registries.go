package cpbootstrap

import (
	"fmt"
	"net/http"
	"slices"

	"connectrpc.com/connect"
)

type rpcRoutes struct {
	owners map[string]string
	paths  []string
	byPath map[string]rpcRoute

	errorNet connect.Interceptor

	drain connect.Interceptor
}

func newRPCRoutes(errorNet, drain connect.Interceptor) *rpcRoutes {
	return &rpcRoutes{
		owners:   map[string]string{},
		paths:    nil,
		byPath:   map[string]rpcRoute{},
		errorNet: errorNet,
		drain:    drain,
	}
}

func (r *rpcRoutes) forModule(module string) RPCRegistrar {
	return moduleRoutes{module: module, routes: r}
}

type rpcRoute struct {
	build        ServiceBuilder
	interceptors []connect.Interceptor
	handler      http.Handler
}

func (r *rpcRoutes) chain(guards, interceptors []connect.Interceptor) connect.HandlerOption {
	chain := append([]connect.Interceptor{r.errorNet}, guards...)
	chain = append(chain, r.drain)
	return connect.WithInterceptors(append(chain, interceptors...)...)
}

func (r *rpcRoutes) mountOn(router *http.ServeMux, middlewares func(http.Handler) http.Handler) {
	for _, path := range r.paths {
		router.Handle(path, middlewares(r.byPath[path].handler))
	}
}

// Each service is built again, with the guards inside the error net: an interceptor is fixed when its handler is built.
func (r *rpcRoutes) mountGuardedOn(
	router *http.ServeMux,
	middlewares func(http.Handler) http.Handler,
	guards ...connect.Interceptor,
) error {
	for _, path := range r.paths {
		route := r.byPath[path]

		rebuilt, handler := route.build(r.chain(guards, route.interceptors))
		if rebuilt != path || handler == nil {
			return fmt.Errorf("the service on %q was built again on %q", path, rebuilt)
		}

		router.Handle(path, middlewares(handler))
	}

	return nil
}

type moduleRoutes struct {
	module string
	routes *rpcRoutes
}

func (m moduleRoutes) Mount(build ServiceBuilder, interceptors ...connect.Interceptor) error {
	if build == nil {
		return fmt.Errorf("module %s mounted a nil service", m.module)
	}

	path, handler := build(m.routes.chain(nil, interceptors))

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
	m.routes.byPath[path] = rpcRoute{build: build, interceptors: interceptors, handler: handler}

	return nil
}

type operatorGuard struct {
	module       string
	interceptors []connect.Interceptor
}

func newOperatorGuard() *operatorGuard {
	return &operatorGuard{module: "", interceptors: nil}
}

func (g *operatorGuard) forModule(module string) OperatorRegistrar {
	return moduleGuard{module: module, guard: g}
}

func (g *operatorGuard) guarded() bool {
	return g.module != ""
}

type moduleGuard struct {
	module string
	guard  *operatorGuard
}

func (m moduleGuard) Guard(interceptors ...connect.Interceptor) error {
	if len(interceptors) == 0 || slices.Contains(interceptors, nil) {
		return fmt.Errorf("module %s guarded the operator listener with no interceptor", m.module)
	}
	if m.guard.guarded() {
		return fmt.Errorf("module %s guards the operator listener, which %s already guards", m.module, m.guard.module)
	}

	m.guard.module = m.module
	m.guard.interceptors = interceptors

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
