package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
	"github.com/raphoester/clickplanet.lol-ops/internal/accesslog"
	"github.com/raphoester/clickplanet.lol-ops/internal/httpapi"
	"github.com/raphoester/clickplanet.lol-ops/internal/journal"
	"github.com/raphoester/clickplanet.lol-ops/internal/sqlquery"
)

const (
	// Longer than the role's own statement_timeout, so postgres is the one that says why.
	statementTimeout = 25 * time.Second
	requestTimeout   = 90 * time.Second
	shutdownTimeout  = 10 * time.Second
)

type Config struct {
	listenAddress      string
	databaseURL        string
	issuer             access.Issuer
	audience           access.Audience
	journalDirectory   string
	accessLogDirectory string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	config, err := loadConfig()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	postgres, err := sqlquery.OpenPostgres(config.databaseURL, statementTimeout)
	if err != nil {
		return fmt.Errorf("failed to set up postgres: %w", err)
	}
	defer func() { _ = postgres.Close() }()

	handler := httpapi.NewHandler(
		access.NewLogged(access.NewCloudflareVerifier(ctx, config.issuer, config.audience), logger),
		sqlquery.NewLogged(postgres, logger),
		journal.NewLogged(journal.NewJournalctl(config.journalDirectory), logger),
		accesslog.NewLogged(accesslog.NewDirectory(config.accessLogDirectory), logger),
	)

	server := &http.Server{
		Addr:              config.listenAddress,
		Handler:           http.TimeoutHandler(handler, requestTimeout, "the request took too long"),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
	}

	failed := make(chan error, 1)
	go func() { failed <- server.ListenAndServe() }()
	logger.Info("listening", slog.String("address", config.listenAddress))

	select {
	case err := <-failed:
		return fmt.Errorf("the server stopped: %w", err)
	case <-ctx.Done():
	}

	closing, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(closing); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to shut down: %w", err)
	}
	return nil
}

func loadConfig() (Config, error) {
	var missing []string
	read := func(name string) string {
		value := os.Getenv(name)
		if value == "" {
			missing = append(missing, name)
		}
		return value
	}

	loaded := Config{
		listenAddress:      read("OPS_LISTEN_ADDRESS"),
		databaseURL:        read("OPS_DATABASE_URL"),
		issuer:             access.Issuer(read("OPS_ACCESS_ISSUER")),
		audience:           access.Audience(read("OPS_ACCESS_AUDIENCE")),
		journalDirectory:   read("OPS_JOURNAL_DIRECTORY"),
		accessLogDirectory: read("OPS_ACCESS_LOG_DIRECTORY"),
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("unset: %v", missing)
	}
	return loaded, nil
}
