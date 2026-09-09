package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/app"
)

const startupTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	a, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	if err := configure(a); err != nil {
		return err
	}

	if err := a.Run(); err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	return nil
}

func configure(a *app.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	if err := a.Configure(ctx); err != nil {
		return fmt.Errorf("failed to configure app: %w", err)
	}

	return nil
}
