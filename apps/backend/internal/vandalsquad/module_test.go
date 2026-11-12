package vandalsquad_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad"
)

func validConfig() vandalsquad.Config {
	return vandalsquad.Config{
		Enabled: true,
		Database: cppg.Config{
			Host: "localhost", Port: "5432", User: "clickplanet", DBName: "clickplanet", SSLMode: "disable",
			Schema: "vandalsquad",
		},
		Redis: cpredis.Config{Address: "localhost:6379"},
	}
}

func TestAConfigWithPostgresAndRedisAloneIsValid(t *testing.T) {
	require.NoError(t, validConfig().Validate(), "the page, the window and the interval have defaults")
}

func TestOffChecksNothing(t *testing.T) {
	assert.NoError(t, vandalsquad.Config{}.Validate())
}

func TestOnWithAPartWrongIsRefused(t *testing.T) {
	for want, change := range map[string]func(*vandalsquad.Config){
		"vandalSquad.database": func(c *vandalsquad.Config) { c.Database.Schema = "" },
		"vandalSquad.redis":    func(c *vandalsquad.Config) { c.Redis.Address = "" },
		"vandalSquad.window":   func(c *vandalsquad.Config) { c.Window = 73 * time.Hour },
		"vandalSquad.gameUrl":  func(c *vandalsquad.Config) { c.GameURL = "http://clickplanet.lol/play" },
		"vandalSquad.interval": func(c *vandalsquad.Config) { c.Window, c.Interval = time.Minute, time.Hour },
	} {
		t.Run(want, func(t *testing.T) {
			config := validConfig()
			change(&config)

			assert.ErrorContains(t, config.Validate(), want)
		})
	}
}
