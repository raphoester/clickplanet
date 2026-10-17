package admingate_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/admingate"
)

func TestNoOriginIsNoAdminPageAndSoNoGate(t *testing.T) {
	gate, err := admingate.New(admingate.Config{}, "127.0.0.1:8082", slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Empty(t, gate.Interceptors, "a gated admin listener then refuses the boot")
}

func TestTheGateAsksAuthOverTheInternalListener(t *testing.T) {
	_, err := admingate.New(admingate.Config{Origin: "https://admin.clickplanet.lol"}, "", slog.New(slog.DiscardHandler))
	require.ErrorContains(t, err, "internalBindAddress")

	gate, err := admingate.New(admingate.Config{Origin: "https://admin.clickplanet.lol"}, "127.0.0.1:8082", slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.NotEmpty(t, gate.Interceptors)
	assert.NotEmpty(t, gate.Middlewares)
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
		err := admingate.Config{Origin: origin}.Validate()
		if ok {
			require.NoError(t, err, origin)
		} else {
			require.ErrorContains(t, err, "adminGate.origin", origin)
		}
	}
}
