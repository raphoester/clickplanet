package planet

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Config struct {
	GameMap      GameMapConfig
	TilesStorage inmemory_tile_storage.Config
	RateLimiter  clicks.ThrottleConfig
	Toll         clicks.TollConfig
	HomeSoil     clicks.HomeSoilConfig
	VPNBlocklist cpipblock.Config
	AntiBot      antibot.Config
	Bonus        bonuses.Config

	ChargeStorage inmemory_charge_storage.Config

	Ledger        ledger.Config
	LedgerStorage inmemory_ledger_storage.Config

	Activity activity.Config

	Auth cpsession.VerifierConfig

	Database cppg.Config
}

type GameMapConfig struct {
	MaxIndex uint32
}

func (c Config) Validate() error {
	if c.GameMap.MaxIndex == 0 {
		return errors.New("gameMap.maxIndex is zero: the map has no tiles")
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	if err := c.Toll.Validate(); err != nil {
		return err
	}

	return errors.Join(c.RateLimiter.Validate(), c.Bonus.Validate(), c.AntiBot.Validate(), c.Activity.Validate())
}
