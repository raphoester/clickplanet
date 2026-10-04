package e2e_test

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func (s gameStack) schema(t *testing.T, name string) *cppg.Postgres {
	t.Helper()

	db := cppg.New(s.postgres.ConfigFor(name))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type keptTake struct {
	tile    int
	country string
	takenAt time.Time
	account sql.NullString
}

func TestADeletedAccountsTakesKeepTheirTileFlagAndTimeAndNameNoAccount(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	before := time.Now().Add(-time.Minute)
	ada.click(1, "fr")
	ada.click(2, "de")

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	planet := game.schema(t, "planet")
	require.Eventually(t, func() bool {
		var anonymous int
		err := planet.QueryRowContext(t.Context(), `SELECT count(*) FROM ledger_takes WHERE account IS NULL`).Scan(&anonymous)
		return err == nil && anonymous == 2
	}, 5*time.Second, 20*time.Millisecond)

	rows, err := planet.QueryContext(t.Context(),
		`SELECT tile, country, taken_at, account::text FROM ledger_takes ORDER BY position`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	var takes []keptTake
	for rows.Next() {
		var take keptTake
		require.NoError(t, rows.Scan(&take.tile, &take.country, &take.takenAt, &take.account))
		takes = append(takes, take)
	}
	require.NoError(t, rows.Err())

	require.Len(t, takes, 2)
	for i, want := range []struct {
		tile    int
		country string
	}{{1, "fr"}, {2, "de"}} {
		assert.Equal(t, want.tile, takes[i].tile)
		assert.Equal(t, want.country, takes[i].country)
		assert.WithinRange(t, takes[i].takenAt, before, time.Now())
		assert.False(t, takes[i].account.Valid, "the take names no account")
	}
}
