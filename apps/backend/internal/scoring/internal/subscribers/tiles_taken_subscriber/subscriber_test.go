package tiles_taken_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/tiles_taken_subscriber"
)

const (
	ada   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	grace = "6f1c2f0e-3a8b-4c5d-9e7f-1a2b3c4d5e6f"
)

var (
	seasonZeroStarts = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	seasonOneStarts  = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	duringSeasonZero = time.Date(2026, 10, 31, 22, 0, 0, 0, time.UTC)
	duringSeasonOne  = seasonOneStarts.Add(time.Hour)
)

func subscriber(t *testing.T, store *inmemory_contribution_store.Store) tiles_taken_subscriber.Subscriber {
	t.Helper()
	shelf := inmemory_book_store.New()
	_, err := shelf.Open(t.Context(), books.NewBook("season-0", seasonZeroStarts))
	require.NoError(t, err)
	_, err = shelf.Close(t.Context(), books.NewBook("season-0", seasonZeroStarts).ClosedAt(seasonOneStarts, ""))
	require.NoError(t, err)
	_, err = shelf.Open(t.Context(), books.NewBook("season-1", seasonOneStarts))
	require.NoError(t, err)
	return tiles_taken_subscriber.New(record_take_usecase.New(shelf, store))
}

func takes(account, country string, at time.Time, position uint64, from, to uint32) *planetv1.TilesTaken {
	tiles := make([]*planetv1.TakenTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.TakenTile{TileId: 40 + index, Index: index})
	}
	return &planetv1.TilesTaken{AccountId: account, Country: country, TakenAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func tally(t *testing.T, store *inmemory_contribution_store.Store, book books.BookID, account string) standings.Tally {
	t.Helper()
	id, err := standings.AccountIDOf(account)
	require.NoError(t, err)
	return store.Tally(book, id)
}

func TestATakeCountsForItsFlagAtTheTimeItWasTaken(t *testing.T) {
	store := inmemory_contribution_store.New()

	err := subscriber(t, store).HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, "fr", duringSeasonZero, 9, 0, 1)})

	require.NoError(t, err)
	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}, TakesBelow: standings.TakeMark{Position: 9, Index: 1}},
		tally(t, store, "season-0", ada))
	assert.True(t, tally(t, store, "season-1", ada).Empty())
}

func TestAnEnclosureToldInPartsAndToldTwiceCountsEachTileOnce(t *testing.T) {
	store := inmemory_contribution_store.New()
	handler := subscriber(t, store)
	enclosure := []*planetv1.TilesTaken{takes(ada, "fr", duringSeasonOne, 9, 0, 3), takes(ada, "fr", duringSeasonOne, 9, 3, 7)}

	for range 2 {
		require.NoError(t, handler.HandleBatch(t.Context(), enclosure))
	}
	for _, part := range enclosure {
		require.NoError(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{part}), "a failed batch is handled again one event at a time")
	}

	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 7}, TakesBelow: standings.TakeMark{Position: 9, Index: 7}},
		tally(t, store, "season-1", ada), "one ledger event in two parts, seven takes")
}

func TestEachAccountAndBookOfABatchCountsItsOwnTakes(t *testing.T) {
	store := inmemory_contribution_store.New()

	err := subscriber(t, store).HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, "fr", duringSeasonZero, 1, 0, 1),
		takes(grace, "de", duringSeasonZero, 2, 0, 5),
		takes(ada, "it", duringSeasonOne, 3, 0, 2),
		takes(grace, "de", duringSeasonOne, 4, 0, 1),
		takes(ada, "fr", duringSeasonOne, 5, 0, 1),
	})

	require.NoError(t, err)
	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}, TakesBelow: standings.TakeMark{Position: 1, Index: 1}},
		tally(t, store, "season-0", ada))
	assert.Equal(t, standings.Tally{Main: "de", Tiles: map[standings.Country]uint64{"de": 5}, TakesBelow: standings.TakeMark{Position: 2, Index: 5}},
		tally(t, store, "season-0", grace))
	assert.Equal(t, standings.Tally{Main: "it", Tiles: map[standings.Country]uint64{"it": 2, "fr": 1}, TakesBelow: standings.TakeMark{Position: 5, Index: 1}},
		tally(t, store, "season-1", ada))
	assert.Equal(t, standings.Tally{Main: "de", Tiles: map[standings.Country]uint64{"de": 1}, TakesBelow: standings.TakeMark{Position: 4, Index: 1}},
		tally(t, store, "season-1", grace))
}

func TestABatchWithAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler := subscriber(t, inmemory_contribution_store.New())
	good := takes(ada, "fr", duringSeasonZero, 1, 0, 1)

	require.ErrorIs(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, takes("", "fr", duringSeasonZero, 2, 0, 1)}),
		standings.ErrInvalidAccount)
	require.ErrorIs(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, takes(ada, "", duringSeasonZero, 2, 0, 1)}),
		standings.ErrNoCountry)
	noTime := takes(ada, "fr", duringSeasonZero, 2, 0, 1)
	noTime.TakenAt = nil
	assert.Error(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, noTime}))
}
