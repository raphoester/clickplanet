package postgres_past_name_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/postgres_past_name_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var (
	ada    = players.AccountID{15: 1}
	bob    = players.AccountID{15: 2}
	monday = time.Date(2026, 11, 16, 20, 0, 0, 123_456_789, time.UTC)
)

func left(t *testing.T, account players.AccountID, name players.Name, at time.Time) pastnames.PastName {
	t.Helper()

	kept, err := pastnames.Snapshot{Account: account, Name: name, LeftAt: at}.Restore()
	require.NoError(t, err)
	return kept
}

func kept(t *testing.T, db *cppg.Postgres, account players.AccountID) []string {
	t.Helper()

	rows, err := db.On(t.Context()).QueryContext(t.Context(),
		`SELECT name FROM past_names WHERE account_id = $1 ORDER BY left_at`, uuid.UUID(account))
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func TestANameHeardTwiceIsKeptOnce(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "player", migrations.FS)
	store := postgres_past_name_store.New(db)

	require.NoError(t, store.Record(t.Context(), left(t, ada, "BraveFox42", monday)))
	require.NoError(t, store.Record(t.Context(), left(t, ada, "BraveFox42", monday)))
	require.NoError(t, store.Record(t.Context(), left(t, ada, "Ada_L", monday.Add(time.Hour))))

	assert.Equal(t, []string{"BraveFox42", "Ada_L"}, kept(t, db, ada))
}

func TestForgettingAnAccountKeepsTheOthers(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "player", migrations.FS)
	store := postgres_past_name_store.New(db)
	require.NoError(t, store.Record(t.Context(), left(t, ada, "BraveFox42", monday)))
	require.NoError(t, store.Record(t.Context(), left(t, bob, "QuietOwl17", monday)))

	require.NoError(t, store.Forget(t.Context(), ada))

	assert.Empty(t, kept(t, db, ada))
	assert.Equal(t, []string{"QuietOwl17"}, kept(t, db, bob))
}
