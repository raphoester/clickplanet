package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/player"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/support"
)

type Config struct {
	HTTPServer cpbootstrap.ServerConfig

	Planet planet.Config `koanf:",squash"`

	Auth     auth.Config
	AntiBot  antibot.Config
	Chat     chat.Config
	Player   player.Config
	Seasons  seasons.Config
	Rules    rules.Config
	Missions missions.Config
	Support  support.Config
	// The numbers of the clips the clips job posted.
	Clips clips.ModuleConfig
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	logger.Debug("config", slog.Any("config", config))

	gate, err := auth.NewAdminGate(config.Auth.Admin, config.HTTPServer.InternalBindAddress, logger)
	if err != nil {
		return fmt.Errorf("failed to build the admin gate: %w", err)
	}

	return cpbootstrap.Run(ctx, cpbootstrap.Options{
		Server:    config.HTTPServer,
		Logger:    logger,
		Modules:   describeModules(config),
		AdminGate: gate,
	})
}

func describeModules(config Config) []cpbootstrap.Module {
	return []cpbootstrap.Module{
		auth.NewModule(config.Auth),
		planet.NewModule(config.Planet),
		antibot.NewModule(config.AntiBot),
		chat.NewModule(config.Chat),
		player.NewModule(config.Player),
		seasons.NewModule(config.Seasons),
		rules.NewModule(config.Rules),
		missions.NewModule(config.Missions),
		support.NewModule(config.Support),
		clips.NewModule(config.Clips),
	}
}

func loadConfig() (Config, error) {
	var config Config
	if err := cpconfigs.Load(&config, cpconfigs.FromFlag()); err != nil {
		return Config{}, fmt.Errorf("failed reading config: %w", err)
	}

	return config, nil
}

func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Planet.Validate(),
		c.AntiBot.Validate(),
		c.Chat.Validate(),
		c.Auth.Validate(),
		c.Player.Validate(),
		c.Seasons.Validate(),
		c.Rules.Validate(),
		c.Missions.Validate(),
		c.Support.Validate(),
		c.Clips.Validate(),
	)
}
