package alerts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

func validConfig() alerts.Config {
	return alerts.Config{
		Enabled:  true,
		Database: cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "alerts"},
		Redis:    cpredis.Config{Address: "localhost:6379"},
		Discord: alerts.DiscordConfig{
			ClientID: "1549822806420885616", ClientSecret: "the-secret", BotToken: "the-bot",
			RedirectURL: "https://clickplanet.lol/auth/callback",
			GuildID:     "1400000000000000001", AlertsChannelID: "1400000000000000002", BattlesChannelID: "1400000000000000003",
		},
	}
}

func TestAWholeConfigIsValid(t *testing.T) {
	require.NoError(t, validConfig().Validate())
}

func TestOffChecksNothing(t *testing.T) {
	assert.NoError(t, alerts.Config{}.Validate())
}

func TestOnWithAPartMissingIsRefused(t *testing.T) {
	for want, change := range map[string]func(*alerts.Config){
		"alerts.database":                 func(c *alerts.Config) { c.Database.Host = "" },
		"alerts.discord.botToken is":      func(c *alerts.Config) { c.Discord.BotToken = "" },
		"alerts.discord.clientSecret is":  func(c *alerts.Config) { c.Discord.ClientSecret = "" },
		"alerts.discord.guildId":          func(c *alerts.Config) { c.Discord.GuildID = "the-server" },
		"alerts.discord.alertsChannelId":  func(c *alerts.Config) { c.Discord.AlertsChannelID = "" },
		"alerts.discord.battlesChannelId": func(c *alerts.Config) { c.Discord.BattlesChannelID = "" },
		"alerts.discord.redirectUrl":      func(c *alerts.Config) { c.Discord.RedirectURL = "/auth/callback" },
		"alerts.gameUrl":                  func(c *alerts.Config) { c.GameURL = "http://clickplanet.lol/play" },
	} {
		t.Run(want, func(t *testing.T) {
			config := validConfig()
			change(&config)

			assert.ErrorContains(t, config.Validate(), want)
		})
	}
}
