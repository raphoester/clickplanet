package antibot

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Config struct {
	Enabled bool

	Database cppg.Config
	Redis    cpredis.Config

	guard.Config `koanf:",squash"`
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("antiBot.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("antiBot.redis: %w", err)
	}

	return c.Config.Validate() //nolint:wrapcheck // it names the bound it refuses.
}
