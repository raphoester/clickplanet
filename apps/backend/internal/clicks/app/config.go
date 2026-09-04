package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/redis_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xredis"
)

type Config struct {
	HTTPServer   HTTPServerConfig
	GameMap      GameMapConfig
	Redis        xredis.Config
	TilesStorage TilesStorageConfig
	Bookkeeper   BookkeeperConfig
}

type HTTPServerConfig struct {
	BindAddress string
	Format      string
}

type GameMapConfig struct {
	MaxIndex uint32
}

// Tile storage drivers. The memory driver keeps the whole map in process and
// persists it to a local snapshot file; the redis driver is the original
// implementation, kept so a rollback is a config change.
const (
	DriverMemory = "memory"
	DriverRedis  = "redis"
)

type TilesStorageConfig struct {
	// Driver selects the tile storage implementation: "memory" (default) or
	// "redis". Redis is only dialled when the redis driver is selected.
	Driver string
	Memory memory_tile_storage.Config
	Redis  redis_tile_storage.Config
}

// BookkeeperConfig controls the background job that reports recent activity to
// X. It used to be the separate cmd/bookkeeper process; it now runs in-process
// so it can read the memory driver's state. Off by default.
type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}
