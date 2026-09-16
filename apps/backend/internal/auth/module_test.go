package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func validConfig() auth.Config {
	secret, _ := cpsession.TestKeyPair()
	config := auth.Config{
		SignerConfig: cpsession.SignerConfig{Enabled: true, Secret: secret},
		Database:     cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "auth"},
	}
	config.SignIn.Enabled = true
	config.SignIn.RedirectURL = "https://clickplanet.lol/auth/callback"
	config.Google.ClientID = "google-client"
	config.Google.ClientSecret = "google-secret"
	return config
}

func TestAConfigWithOneProviderIsValid(t *testing.T) {
	require.NoError(t, validConfig().Validate())
}

func TestSignInOffChecksNoProvider(t *testing.T) {
	config := auth.Config{SignerConfig: validConfig().SignerConfig, Database: validConfig().Database}

	assert.NoError(t, config.Validate())
}

func TestSignInWithNoProviderIsRefused(t *testing.T) {
	config := validConfig()
	config.Google.ClientID = ""

	assert.ErrorContains(t, config.Validate(), "auth.signIn.enabled is true and no provider has a clientId")
}

func TestAProviderWithoutItsSecretIsRefused(t *testing.T) {
	config := validConfig()
	config.Discord.ClientID = "discord-client"

	assert.ErrorContains(t, config.Validate(), "auth.discord.clientSecret is empty while auth.discord.clientId is set")
}

func TestARedirectThatIsNotAnAbsoluteURLIsRefused(t *testing.T) {
	for name, redirect := range map[string]string{
		"empty":      "",
		"relative":   "/auth/callback",
		"with query": "https://clickplanet.lol/auth/callback?next=/",
	} {
		t.Run(name, func(t *testing.T) {
			config := validConfig()
			config.SignIn.RedirectURL = redirect

			assert.ErrorContains(t, config.Validate(), "auth.signIn.redirectUrl")
		})
	}
}

func TestThePruneMayNotDeleteAGuestWhoseCookieIsStillLive(t *testing.T) {
	config := validConfig()
	config.Sessions.GuestTTL = 120 * 24 * time.Hour
	require.NoError(t, config.Validate(), "left unset, the prune waits out the guest lifetime")

	config.Prune.IdleFor = 90 * 24 * time.Hour
	assert.ErrorContains(t, config.Validate(), "auth.prune.idleFor 2160h0m0s is shorter than auth.sessions.guestTTL 2880h0m0s")
}
