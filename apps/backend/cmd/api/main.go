// api is the composition root: the config, what two contexts share, the module list.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Config struct {
	HTTPServer cpbootstrap.ServerConfig

	// Squashed: the planet keys sit at the top level of the file.
	Planet planet.Config `koanf:",squash"`

	Auth auth.Config
	Chat chat.Config
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	config, err := loadConfig(cpconfigs.FromFlag())
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // todo: inject config
	}))
	logger.Debug("config", slog.Any("config", config))

	return cpbootstrap.Run(ctx, cpbootstrap.Options{
		Server:  config.HTTPServer,
		Logger:  logger,
		Modules: describeModules(config),
	})
}

// describeModules is the whole aggregation: every module takes its own config
// and builds everything else itself.
func describeModules(config Config) []cpbootstrap.Module {
	return []cpbootstrap.Module{
		auth.NewModule(config.Auth),
		planet.NewModule(config.Planet),
		chat.NewModule(config.Chat),
	}
}

// loadConfig takes where the file comes from, so the derivation below is exercised
// by a test rather than only by a running binary.
func loadConfig(from cpconfigs.LoadOption) (Config, error) {
	var config Config
	if err := cpconfigs.Load(&config, from); err != nil {
		return Config{}, fmt.Errorf("failed reading config: %w", err)
	}

	// The one thing neither module can do for itself: planet verifies with the
	// public half of the seed only auth is given, so the composition root — which
	// is the only part that sees both blocks — derives it. There is one key in the
	// file, and nothing to keep in step with it.
	if config.Auth.Enabled {
		public, err := cpsession.PublicKeyOf(config.Auth.Secret)
		if err != nil {
			return Config{}, fmt.Errorf("failed deriving the click token verifying key: %w", err)
		}
		config.Planet.Auth.PublicKey = public
	}

	return config, nil
}

// Validate asks each block to check itself, and reports everything wrong at once.
func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Planet.Validate(),
		c.Chat.Validate(),
		c.Auth.Validate(),
	)
}
