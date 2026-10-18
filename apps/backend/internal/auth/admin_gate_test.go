package auth_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
)

func TestNoOriginIsNoAdminPageAndSoNoGate(t *testing.T) {
	gate, err := auth.NewAdminGate(auth.AdminConfig{}, "127.0.0.1:8082", slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Empty(t, gate.Interceptors, "an admin listener then refuses the boot")
}

func TestTheGateAsksAuthOverTheInternalListener(t *testing.T) {
	_, err := auth.NewAdminGate(auth.AdminConfig{Origin: "https://admin.clickplanet.lol"}, "", slog.New(slog.DiscardHandler))
	require.ErrorContains(t, err, "internalBindAddress")

	gate, err := auth.NewAdminGate(auth.AdminConfig{Origin: "https://admin.clickplanet.lol"}, "127.0.0.1:8082", slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.NotEmpty(t, gate.Interceptors)
	assert.NotEmpty(t, gate.Middlewares)
}

func adminPage(origin string) auth.AdminConfig {
	config := auth.AdminConfig{Origin: origin, RedirectURL: "https://admin.clickplanet.lol/auth/callback"}
	config.Google.ClientID = "admin-client"
	config.Google.ClientSecret = "admin-secret"
	return config
}

func TestTheOriginIsExact(t *testing.T) {
	for origin, ok := range map[string]bool{
		"":                               true,
		"https://admin.clickplanet.lol":  true,
		"http://localhost:5174":          true,
		"*":                              false,
		"https://admin.clickplanet.lol/": false,
		"admin.clickplanet.lol":          false,
	} {
		err := adminPage(origin).Validate()
		if ok {
			require.NoError(t, err, origin)
		} else {
			require.ErrorContains(t, err, "auth.admin.origin", origin)
		}
	}
}

func TestAnAdminPageSignsInWithItsOwnGoogleClient(t *testing.T) {
	noSecret := adminPage("https://admin.clickplanet.lol")
	noSecret.Google.ClientSecret = ""
	require.ErrorContains(t, noSecret.Validate(), "auth.admin.google")

	noRedirect := adminPage("https://admin.clickplanet.lol")
	noRedirect.RedirectURL = "/auth/callback"
	require.ErrorContains(t, noRedirect.Validate(), "auth.admin.redirectUrl")

	require.NoError(t, auth.AdminConfig{}.Validate(), "no admin page asks for nothing")
}

func TestAnAdminGateNeedsAuthOn(t *testing.T) {
	off := auth.Config{Admin: auth.AdminConfig{Origin: "https://admin.clickplanet.lol"}}

	require.ErrorContains(t, off.Validate(), "auth.admin.origin", "the gate asks auth who operates")
	require.NoError(t, auth.Config{}.Validate())
}
