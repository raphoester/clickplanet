// Package app is the composition root: the config, what two contexts share, and the module list.
package app

import (
	"context"
	"flag"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/in_memory_country_checker"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/cfgutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	kernelsession "github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
	"github.com/raphoester/clickplanet.lol-backend/internal/session"
)

func Run(ctx context.Context) error {
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
		BindAddress: config.HTTPServer.BindAddress,
		Logger:      logger,
		Modules:     modules,
	})
}

// describeModules builds what more than one context needs, then names the modules.
func describeModules(config Config, logger logging.Logger) ([]bootstrap.Module, error) {
	countries := in_memory_country_checker.New()

	// Sessions mint what clicks verifies; a nil signer means sessions are off.
	var signer *kernelsession.Signer

	modules := make([]bootstrap.Module, 0, 3)

	if config.Session.Enabled {
		built, err := session.NewSigner(config.Session, logger)
		if err != nil {
			return nil, err
		}
		signer = built

		modules = append(modules, session.NewModule(config.Session, signer))
	}

	modules = append(modules, clicks.NewModule(config.Clicks, clicks.Deps{
		Countries:       countries,
		Signer:          signer,
		EnforceSessions: config.Session.Enforce,
		StreamHeartbeat: config.HTTPServer.StreamHeartbeat,
	}))

	if config.Chat.Enabled {
		modules = append(modules, chat.NewModule(config.Chat, chat.Deps{
			Countries:       countries,
			StreamHeartbeat: config.HTTPServer.StreamHeartbeat,
		}))
	}

	return modules, nil
}

func loadConfig() (Config, error) {
	path := flag.String("config", "", "path to config file")
	flag.Parse()

	var config Config
	if err := cfgutil.NewLoader(*path).Unmarshal(&config); err != nil {
		return Config{}, fmt.Errorf("failed reading config: %w", err)
	}

	return config, nil
}
