package quizzes

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// The quizzes' rules are not here: they are the rule sets postgres keeps (quizzes.rule_sets), edited from the
// backoffice and played by the schedule.
type Config struct {
	Database cppg.Config
	Redis    cpredis.Config

	// How long a caller with no stream open is remembered, and how often the quizzes look for who is due.
	Offers quizzes.Config
}

func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("quizzes.database: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("quizzes.redis: %w", err)
	}
	return nil
}
