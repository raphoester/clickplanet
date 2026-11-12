package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestTheOutboxKeepsADiscordJoinGranted(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "auth", migrations.FS)

	_, err := db.On(t.Context()).ExecContext(t.Context(),
		`INSERT INTO outbox (event, payload) VALUES ('auth.v1.DiscordJoinGranted', '')`)

	require.NoError(t, err)
}
