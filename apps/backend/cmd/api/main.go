// api is the composition root: the config, what two contexts share, the module list.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/configs"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/logging/lf"
)

type Config struct {
	HTTPServer bootstrap.ServerConfig

	// Squashed: the planet keys sit at the top level of the file.
	Planet planet.Config `koanf:",squash"`

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
		planet.NewModule(config.Planet),
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

// Validate asks each block to check itself, and reports everything wrong at once.
func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Planet.Validate(),
		c.Session.Validate(),
		c.Chat.Validate(),
	)
}
