package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
)

type Config struct {
	HTTPServer   HTTPServerConfig
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	Bookkeeper   BookkeeperConfig
}

type HTTPServerConfig struct {
	BindAddress string
}

type GameMapConfig struct {
	MaxIndex uint32
}

// BookkeeperConfig controls the background job that reports recent activity to
// X. It used to be the separate cmd/bookkeeper process; it now runs in-process
// so it can read the tile storage's recent updates. Off by default.
type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}
