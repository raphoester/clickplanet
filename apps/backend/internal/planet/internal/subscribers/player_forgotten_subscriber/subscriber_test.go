package player_forgotten_subscriber_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/postgres_ledger_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/anonymize_takes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/player_forgotten_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	gone = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	kept = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
)

var at = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountIsTakenOffEveryTakeItMade(t *testing.T) {
	ctx := t.Context()
	db := cppg.StartTestServer(t).OpenSchema(t, "planet", migrations.FS)
	store := postgres_ledger_store.New(db)
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, store, slog.New(slog.DiscardHandler))
	require.NoError(t, takings.Load(ctx))

	takings.Append(ledger.Taking{Tile: 1, Scope: "1.2.3.4", Account: gone, Country: "fr", At: at})
	takings.Append(ledger.Taking{Tile: 2, Scope: "1.2.3.4", Account: kept, Country: "fr", At: at.Add(time.Hour)})
	require.NoError(t, takings.Flush(ctx))
	takings.ForgetBefore(at.Add(time.Minute))
	require.NoError(t, takings.Flush(ctx))
	takings.Append(ledger.Taking{Tile: 3, Scope: "1.2.3.4", Account: gone, Country: "de", Previous: "fr", At: at.Add(2 * time.Hour)})

	err := player_forgotten_subscriber.New(anonymize_takes_usecase.New(takings, store, arenas.NewMemoryStore(), tiles())).
		Handle(ctx, &playerv1.PlayerForgotten{AccountId: gone})
	require.NoError(t, err)

	var all, named, held int
	require.NoError(t, db.On(ctx).QueryRowContext(ctx,
		`SELECT count(*), count(account), count(*) FILTER (WHERE account = $1) FROM ledger_events`,
		uuid.MustParse(gone)).Scan(&all, &named, &held))
	assert.Equal(t, 3, all, "every take is still there, the one behind the head too")
	assert.Equal(t, 1, named, "only the other account's take names an account")
	assert.Zero(t, held)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	persistence := inmemory_ledger_storage.NewMemoryPersistence()
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, persistence, slog.New(slog.DiscardHandler))

	err := player_forgotten_subscriber.New(anonymize_takes_usecase.New(takings, persistence, arenas.NewMemoryStore(), tiles())).
		Handle(t.Context(), &playerv1.PlayerForgotten{})

	assert.ErrorIs(t, err, ledger.ErrInvalidAccount)
}

func tiles() *inmemory_tile_storage.Storage {
	return inmemory_tile_storage.New(clicks.BordersOf(10), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}), cptime.SystemClock{}, slog.New(slog.DiscardHandler))
}
