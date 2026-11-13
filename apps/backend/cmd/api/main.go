package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/player"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/support"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad"
)

type Config struct {
	HTTPServer cpbootstrap.ServerConfig

	Planet planet.Config `koanf:",squash"`

	Auth        auth.Config
	AntiBot     antibot.Config
	Chat        chat.Config
	Player      player.Config
	Progression progression.Config
	Scoring     scoring.Config
	Gamemaster  gamemaster.Config
	Missions    missions.Config
	// The boxes, and what a box, a right answer and a sign-in grant.
	Bonuses bonuses.Config
	// The quizzes, the bank and how hard each question is.
	Quizzes quizzes.Config
	Support support.Config
	MCP     mcp.Config
	// Who each clip names and who keeps out, the names a post may show, and the operator's videos.
	Clips clips.ModuleConfig
	// The game's Discord server: a player's link, its flag's role, and the pings.
	Alerts alerts.Config
	// A battle's teaser: each rule set's cards, over the backgrounds the operator uploads.
	Promo promo.Config
	// The game's posts on its own TikTok and YouTube: the calendar they keep, and the posting service.
	Publishing publishing.Config
	// The squad that flags the swastikas players paint on the map.
	VandalSquad vandalsquad.Config
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
		bonuses.NewModule(config.Bonuses),
		quizzes.NewModule(config.Quizzes),
		antibot.NewModule(config.AntiBot),
		chat.NewModule(config.Chat),
		player.NewModule(config.Player),
		progression.NewModule(config.Progression),
		scoring.NewModule(config.Scoring),
		gamemaster.NewModule(config.Gamemaster),
		missions.NewModule(config.Missions),
		support.NewModule(config.Support),
		clips.NewModule(config.Clips),
		alerts.NewModule(config.Alerts),
		promo.NewModule(config.Promo),
		publishing.NewModule(config.Publishing),
		vandalsquad.NewModule(config.VandalSquad),
		mcp.NewModule(config.MCP),
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
		c.Progression.Validate(),
		c.Scoring.Validate(),
		c.Gamemaster.Validate(),
		c.Missions.Validate(),
		c.Bonuses.Validate(),
		c.Quizzes.Validate(),
		c.Support.Validate(),
		c.Clips.Validate(),
		c.Alerts.Validate(),
		c.Promo.Validate(),
		c.Publishing.Validate(),
		c.VandalSquad.Validate(),
		c.MCP.Validate(),
	)
}
