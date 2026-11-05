package tile_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/tile_taken_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func subscriber(t *testing.T, store *inmemory_contribution_store.Store) tile_taken_subscriber.Subscriber {
	t.Helper()
	shelf := inmemory_book_store.New()
	_, err := shelf.Open(t.Context(), books.NewBook("season-0", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, err)
	return tile_taken_subscriber.New(record_take_usecase.New(shelf, store))
}

func TestATakeCountsForItsFlagAtTheTimeItWasTaken(t *testing.T) {
	store := inmemory_contribution_store.New()

	err := subscriber(t, store).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(seasonZeroEnds.Add(-time.Hour)), Position: 9,
	})

	require.NoError(t, err)
	id, err := standings.AccountIDOf(account)
	require.NoError(t, err)
	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}, TakesBelow: standings.TakeMark{Position: 9, Index: 1}}, store.Tally("season-0", id))
}

func TestASpreadCountsEachTileItTookOnce(t *testing.T) {
	store := inmemory_contribution_store.New()
	handler := subscriber(t, store)
	spread := make([]*planetv1.TileTaken, 0, 7)
	for index := range uint32(7) {
		spread = append(spread, &planetv1.TileTaken{
			AccountId: account, TileId: 40 + index, Country: "fr", TakenAt: timestamppb.New(seasonZeroEnds.Add(-time.Hour)),
			Position: 9, Index: index,
		})
	}

	for range 2 {
		for _, take := range spread {
			require.NoError(t, handler.Handle(t.Context(), take))
		}
	}

	id, err := standings.AccountIDOf(account)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), store.Tally("season-0", id).Tiles["fr"], "one ledger event, seven takes, told twice")
}

func TestAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler := subscriber(t, inmemory_contribution_store.New())
	at := timestamppb.New(seasonZeroEnds.Add(-time.Hour))

	require.ErrorIs(t, handler.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, Country: "fr", TakenAt: at}),
		standings.ErrInvalidAccount)
	require.ErrorIs(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: at}),
		standings.ErrNoCountry)
	assert.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
}
