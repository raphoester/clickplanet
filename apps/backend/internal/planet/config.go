package planet

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Config is squashed into the process config, so these keys sit at the top
// level of the file where they have always been.
type Config struct {
	GameMap      GameMapConfig
	TilesStorage inmemory_tile_storage.Config
	RateLimiter  cpratelimit.Config
	Toll         clicks.TollConfig
	VPNBlocklist cpipblock.Config
	AntiBot      antibot.Config
	Bonus        bonuses.Config

	// Who last took each tile, for the operator tools.
	Ledger        ledger.Config
	LedgerStorage inmemory_ledger_storage.Config

	// The verifying half of the `auth:` block: a public key, never the seed.
	Auth cpsession.VerifierConfig

	Database cppg.Config
}

type GameMapConfig struct {
	MaxIndex uint32
}

// Validate refuses a map of no tiles, which would refuse every click. The
// `auth:` block it reads is the auth context's to check.
func (c Config) Validate() error {
	if c.GameMap.MaxIndex == 0 {
		return errors.New("gameMap.maxIndex is zero: the map has no tiles")
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	if err := c.Toll.Validate(c.RateLimiter.Capacity()); err != nil {
		return err
	}

	return errors.Join(c.Bonus.Validate(), c.AntiBot.Validate())
}
