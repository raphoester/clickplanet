// api is the composition root: the config, what two contexts share, the module list.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/configs"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/countries"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/session"
)

type Config struct {
	HTTPServer bootstrap.ServerConfig

	// Squashed: the clicks keys sit at the top level of the file.
	Clicks clicks.Config `koanf:",squash"`

	Session session.Config
	Chat    chat.Config
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	logger := logging.NewSLogger() // todo: inject config
	logger.Debug("config", lf.Any("config", config))

	modules, err := describeModules(config, logger)
	if err != nil {
		return err
	}

	return bootstrap.Run(ctx, bootstrap.Options{
		Server:  config.HTTPServer,
		Logger:  logger,
		Modules: modules,
	})
}

// describeModules builds what more than one context needs, then lists them all.
func describeModules(config Config, logger logging.Logger) ([]bootstrap.Module, error) {
	// Sessions mint what clicks verifies; nil when sessions are off.
	signer, err := session.NewSigner(config.Session, logger)
	if err != nil {
		return nil, err
	}

	// The country list. Both the tile game and the chat validate against it.
	checker := countries.New()

	return []bootstrap.Module{
		session.NewModule(config.Session, signer),
		clicks.NewModule(config.Clicks, clicks.Deps{
			Countries:       checker,
			Signer:          signer,
			EnforceSessions: config.Session.Enforce,
			Server:          config.HTTPServer,
		}),
		chat.NewModule(config.Chat, chat.Deps{
			Countries: checker,
			Server:    config.HTTPServer,
		}),
	}, nil
}

func loadConfig() (Config, error) {
	var config Config
	if err := configs.Load(&config, configs.FromFlag()); err != nil {
		return Config{}, fmt.Errorf("failed reading config: %w", err)
	}

	return config, nil
}

// Validate refuses the two settings that have no usable zero value: an empty
// address listens on port 80, and a map of no tiles refuses every click.
func (c Config) Validate() error {
	if c.HTTPServer.BindAddress == "" {
		return errors.New("httpServer.bindAddress is empty")
	}
	if c.Clicks.GameMap.MaxIndex == 0 {
		return errors.New("gameMap.maxIndex is zero: the map has no tiles")
	}

	return nil
}
