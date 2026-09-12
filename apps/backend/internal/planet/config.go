package planet

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Config is squashed into the process config, so these keys sit at the top
// level of the file where they have always been.
type Config struct {
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	RateLimiter  cpratelimit.Config
	VPNBlocklist cpipblock.Config
	AntiBot      antibot.Config
	Bonus        bonus.Config

	// The same `session:` keys the session context mints with. Declared here
	// rather than handed over, so this module needs nothing but its config.
	Session cpsession.Config
}

type GameMapConfig struct {
	MaxIndex uint32
}

// Validate refuses a map of no tiles, which would refuse every click. The
// `session:` block it reads is the session context's to check, and does not
// exist without it.
func (c Config) Validate() error {
	if c.GameMap.MaxIndex == 0 {
		return errors.New("gameMap.maxIndex is zero: the map has no tiles")
	}

	return nil
}
