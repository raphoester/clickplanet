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

	return bootstrap.Run(ctx, bootstrap.Options{
		Server:  config.HTTPServer,
		Logger:  logger,
		Modules: describeModules(config),
	})
}

// describeModules is the whole aggregation: every module takes its own config
// and builds everything else itself.
func describeModules(config Config) []bootstrap.Module {
	return []bootstrap.Module{
		session.NewModule(config.Session),
		clicks.NewModule(config.Clicks),
		chat.NewModule(config.Chat),
	}
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

	// Both contexts derive their signer from this one string, so a server that
	// invented one could not verify what it had just minted.
	if c.Session.Enabled && c.Session.Secret == "" {
		return errors.New("session.secret is empty while session.enabled is true: set SESSION_SECRET")
	}

	return nil
}
