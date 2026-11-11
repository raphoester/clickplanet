package bonuses

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// The offers' rules are not here: they are the rule sets postgres keeps (bonuses.rule_sets), edited from the
// backoffice and played by the schedule.
type Config struct {
	Database cppg.Config
	Redis    cpredis.Config

	// How long a caller with no stream open is remembered, and how often the offers look for who is due.
	Offers bonuses.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("bonuses.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("bonuses.redis: %w", err)
	}
	return nil
}
