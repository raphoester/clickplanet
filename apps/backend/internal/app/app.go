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
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
)

type App struct {
	config Config
	logger logging.Logger
	server *http.Server

	ctx    context.Context
	cancel context.CancelFunc

	runners       []func()
	shutdownFuncs []func()

	rpcServices []rpcService

	promRegistry *prometheus.Registry

	// Left by the session context for the clicks one to verify against. Nil
	// when sessions are disabled, which is what leaves the click chain as it
	// was before they existed.
	sessionSigner *session.Signer
}

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
	// Before the clicks context: it verifies what this one mints.
	if err := a.configureSessionIfEnabled(ctx); err != nil {
		return fmt.Errorf("failed to configure the session context: %w", err)
	}

	if err := a.configureClicks(ctx); err != nil {
		return fmt.Errorf("failed to configure the clicks context: %w", err)
	}

	if err := a.configureChatIfEnabled(ctx); err != nil {
		return fmt.Errorf("failed to configure the chat context: %w", err)
	}

	rpcMiddlewares := httpserver.MiddlewareStack(
		httpserver.NewLoggingMiddleware(a.logger),
		httpserver.IPReaderMiddleware,
		httpserver.CorsMiddleware,
	)

	router := http.NewServeMux()

	for _, service := range a.rpcServices {
		router.Handle(service.path, rpcMiddlewares(service.handler))
	}

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

func (a *App) mountRPC(path string, handler http.Handler) {
	a.rpcServices = append(a.rpcServices, rpcService{path: path, handler: handler})
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
