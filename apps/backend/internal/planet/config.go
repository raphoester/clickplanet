package planet

import (
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// The game's rules are not here: they are the rule set postgres keeps (planet.rule_sets), edited from the backoffice.
type Config struct {
	TilesStorage inmemory_tile_storage.Config
	RateLimiter  RateLimiterConfig
	VPNBlocklist cpipblock.Config

	Ledger        ledger.Config
	LedgerStorage inmemory_ledger_storage.Config

	Auth cpsession.VerifierConfig

	Database cppg.Config
	Redis    cpredis.Config
}

type RateLimiterConfig struct {
	SweepInterval time.Duration
}

func (c Config) Validate() error {
	if err := c.Ledger.Validate(); err != nil {
		return fmt.Errorf("ledger: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	return nil
}
