package postgres_ledger_players_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/postgres_ledger_players"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	ada  = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob  = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
	cleo = "9e1f2a3b-4c5d-4e6f-8a7b-0c1d2e3f4a5b"
	dan  = "c3d4e5f6-0718-4293-a4b5-c6d7e8f90a1b"
)

var start = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func record(t *testing.T, db *cppg.Postgres, events ...ledger.Event) {
	t.Helper()

	stored := make([]inmemory_ledger_storage.Stored, 0, len(events))
	for i, event := range events {
		entry, err := event.Entry()
		require.NoError(t, err)
		stored = append(stored, inmemory_ledger_storage.Stored{Position: ledger.Position(i), Entry: entry}) //nolint:gosec // a few events.
	}
	require.NoError(t, postgres_ledger_store.New(db).Save(t.Context(), inmemory_ledger_storage.Changes{Entries: slices.Values(stored)}))
}

func TestThePlayersAreEveryAccountThatActedBeforeTheStartInAccountOrder(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	record(t, db,
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: cleo, Country: "fr", At: start.Add(-time.Hour)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.7", Account: ada, Country: "fr", At: start.Add(-time.Minute)},
		ledger.Taking{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "fr", At: start.Add(-time.Second)},
		ledger.Striking{Tile: 4, Scope: "203.0.113.8", Account: bob, Country: "de", At: start.Add(-time.Millisecond)},
		ledger.Taking{Tile: 5, Scope: "203.0.113.9", Account: dan, Country: "dz", At: start},
		ledger.Taking{Tile: 6, Scope: "203.0.113.9", Country: "dz", At: start.Add(-time.Hour)},
	)
	players := postgres_ledger_players.New(db)

	first, err := players.Players(t.Context(), fills.Page{Before: start, Size: 2})
	require.NoError(t, err)
	assert.Equal(t, []bonuses.Holder{ada, bob}, first, "each account once, a strike as a take")

	rest, err := players.Players(t.Context(), fills.Page{Before: start, After: bob, Size: 2})
	require.NoError(t, err)
	assert.Equal(t, []bonuses.Holder{cleo}, rest, "an account that first played at the start is not one, nor a take with none")
}

func TestADeletedAccountIsNoLongerAPlayer(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	record(t, db,
		ledger.Taking{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "fr", At: start.Add(-time.Hour)},
		ledger.Taking{Tile: 2, Scope: "203.0.113.8", Account: bob, Country: "de", At: start.Add(-time.Hour)},
	)
	id, err := ledger.AccountIDOf(ada)
	require.NoError(t, err)
	require.NoError(t, postgres_ledger_store.New(db).AnonymizeTakes(t.Context(), id))

	players, err := postgres_ledger_players.New(db).Players(t.Context(), fills.Page{Before: start, Size: 500})

	require.NoError(t, err)
	assert.Equal(t, []bonuses.Holder{bob}, players)
}
