package bonuses

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// The bonuses module's rules are not here: they are the rule sets postgres keeps (bonuses.rule_sets), edited from the
// backoffice and played by the schedule.
type Config struct {
	Database cppg.Config
	Redis    cpredis.Config
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
