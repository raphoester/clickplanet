package inmemory_spent_codes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/inmemory_spent_codes"
)

func TestACodeIsSpentOnceAndForgottenWhenItLapses(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	codes := inmemory_spent_codes.New()

	require.NoError(t, codes.Spend(code(t, "code 1", now.Add(time.Minute)), now))
	require.ErrorIs(t, codes.Spend(code(t, "code 1", now.Add(time.Minute)), now.Add(time.Second)), connections.ErrInvalidGrant)
	require.NoError(t, codes.Spend(code(t, "code 2", now.Add(time.Minute)), now), "another code")

	require.NoError(t, codes.Spend(code(t, "code 1", now.Add(2*time.Minute)), now.Add(time.Minute)),
		"forgotten once lapsed: the code itself refuses it by then")
}

func code(t *testing.T, id connections.CodeID, expiresAt time.Time) *connections.Code {
	t.Helper()
	code, err := connections.CodeSnapshot{ID: id, ClientID: "the client", Challenge: "the challenge", ExpiresAt: expiresAt}.Restore()
	require.NoError(t, err)
	return code
}
