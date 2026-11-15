package boxes

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// The boxes' rules are not here: they are the rule sets postgres keeps (boxes.rule_sets), edited from the backoffice
// and played by the schedule.
type Config struct {
	Database cppg.Config
	Redis    cpredis.Config

	// How long a caller with no stream open is remembered, and how often the boxes look for who is due.
	Offers boxes.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("boxes.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("boxes.redis: %w", err)
	}
	return nil
}
