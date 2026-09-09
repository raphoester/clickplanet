package app

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/prom"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/cfgutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

// App is the composition root. It is the only place that knows about more than
// one bounded context: clicks and chat are wired side by side here and share
// nothing but this file's imports, the process, and the transport.
type App struct {
	config Config
	logger logging.Logger
	server *http.Server

	ctx    context.Context
	cancel context.CancelFunc

	runners       []func()
	shutdownFuncs []func()

	rpcServices []rpcService
	wsRoutes    []func(*http.ServeMux)

	promRegistry *prometheus.Registry
}

// rpcService is a Connect handler and the path Connect derived for it from its
// proto package. Each bounded context contributes its own.
type rpcService struct {
	path    string
	handler http.Handler
}

func New() (*App, error) {
	c := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg := Config{}
	if err := cfgutil.NewLoader(*c).Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed reading config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		config: cfg,
		logger: logging.NewSLogger(), // todo: inject config
		ctx:    ctx,
		cancel: cancel,
	}

	app.logger.Debug("config", lf.Any("config", cfg))
	return app, nil
}

func (a *App) Configure(ctx context.Context) error {
	if err := a.configureClicks(ctx); err != nil {
		return fmt.Errorf("failed to configure the clicks context: %w", err)
	}

	rpcMiddlewares := httpserver.MiddlewareStack(
		httpserver.NewLoggingMiddleware(a.logger),
		httpserver.IPReaderMiddleware,
		httpserver.CorsMiddleware,
	)

	wsMiddlewares := httpserver.MiddlewareStack(
		httpserver.IPReaderMiddleware,
	)

	router := http.NewServeMux()

	// Connect names its own path from the proto package, so the two services
	// land on /planet.v1.ClickService/ and /chat.v1.ChatService/ with no
	// prefix of ours in front of either.
	for _, service := range a.rpcServices {
		router.Handle(service.path, rpcMiddlewares(service.handler))
	}

	wsRouter := http.NewServeMux()
	for _, declare := range a.wsRoutes {
		declare(wsRouter)
	}
	router.Handle("/ws/", http.StripPrefix("/ws", wsMiddlewares(wsRouter)))

	a.declarePrometheusRoutes(router)

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	a.server = &http.Server{
		Addr:      a.config.HTTPServer.BindAddress,
		Handler:   router,
		Protocols: protocols,
	}

	return nil
}

// mountRPC registers a bounded context's Connect handler. Called from wiring,
// not from a context: nothing under internal/clicks or internal/chat knows a
// router exists.
func (a *App) mountRPC(path string, handler http.Handler) {
	a.rpcServices = append(a.rpcServices, rpcService{path: path, handler: handler})
}

// mountWS registers a websocket route under the /ws prefix.
func (a *App) mountWS(declare func(*http.ServeMux)) {
	a.wsRoutes = append(a.wsRoutes, declare)
}

func (a *App) declarePrometheusRoutes(router *http.ServeMux) {
	promRouter := http.NewServeMux()

	a.configurePromRegistryIfNeeded()
	middlewareStack := httpserver.MiddlewareStack(
		httpserver.NewLoggingMiddleware(a.logger),
	)

	promHandler := prom.HandlerForRegistry(a.promRegistry)
	promRouter.HandleFunc("GET /", promHandler.ServeHTTP)

	router.Handle("/metrics", middlewareStack(promRouter))
}
