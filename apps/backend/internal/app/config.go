package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
)

type Config struct {
	HTTPServer   HTTPServerConfig
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	Bookkeeper   BookkeeperConfig
	RateLimiter  ratelimit.Config
	VPNBlocklist ipblock.Config
}

type HTTPServerConfig struct {
	BindAddress string
}

type GameMapConfig struct {
	MaxIndex uint32
}

type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}
