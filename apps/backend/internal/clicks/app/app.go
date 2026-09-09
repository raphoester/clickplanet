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

type App struct {
	config Config
	logger logging.Logger
	server *http.Server

	// ctx spans the app's lifetime: cancelling it stops the background
	// runners, which is what triggers the final tile snapshot.
	ctx    context.Context
	cancel context.CancelFunc

	runners       []func()
	shutdownFuncs []func()

	answerer *httpserver.Answerer
	reader   httpserver.Reader

	promRegistry *prometheus.Registry
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
	app, err := a.configureApp(ctx)
	if err != nil {
		return fmt.Errorf("failed to configure app: %w", err)
	}

	rpcMiddlewares := httpserver.MiddlewareStack(
		httpserver.NewLoggingMiddleware(a.logger),
		httpserver.IPReaderMiddleware,
		httpserver.CorsMiddleware,
	)

	wsMiddlewares := httpserver.MiddlewareStack(
		//httpserver.NewLoggingMiddleware(a.logger), // TODO: fix hijacker problem
		httpserver.IPReaderMiddleware,
	)

	router := http.NewServeMux()

	// Connect names its own path, so it needs no prefix of ours and cannot
	// collide with the deprecated tree below.
	router.Handle(app.connectPath, rpcMiddlewares(app.connectHandler))

	wsRouter := http.NewServeMux()
	app.declareWSRoutes(wsRouter)
	router.Handle("/ws/", http.StripPrefix("/ws", wsMiddlewares(wsRouter)))

	// Deprecated: mounted only until the deployed frontends move over.
	legacyRPCRouter := http.NewServeMux()
	app.declareLegacyRoutes(legacyRPCRouter)
	router.Handle("/v2/rpc/", http.StripPrefix("/v2/rpc", rpcMiddlewares(legacyRPCRouter)))

	legacyWSRouter := http.NewServeMux()
	app.declareWSRoutes(legacyWSRouter)
	router.Handle("/v2/ws/", http.StripPrefix("/v2/ws", wsMiddlewares(legacyWSRouter)))

	a.declarePrometheusRoutes(router)

	// Connect handlers are plain http.Handlers, so everything shares one mux
	// and one server. Unencrypted HTTP/2 is enabled because the generated
	// handler also speaks gRPC and gRPC-Web, and those need it; browsers reach
	// the same routes over HTTP/1.1.
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

type ConfigureAppResponse struct {
	declareWSRoutes     func(mux *http.ServeMux)
	declareLegacyRoutes func(mux *http.ServeMux)
	connectPath         string
	connectHandler      http.Handler
}
