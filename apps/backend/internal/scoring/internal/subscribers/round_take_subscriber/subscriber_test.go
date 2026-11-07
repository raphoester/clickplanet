package round_take_subscriber_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/count_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/round_take_subscriber"
)

const (
	ada   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	grace = "6f1c2f0e-3a8b-4c5d-9e7f-1a2b3c4d5e6f"
)

var (
	day = books.Round{
		ID: "season-0/day-24", Book: "season-0", Times: 1,
		StartsAt: time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC),
	}
	battle = books.Round{
		ID: "0192f5a0-7c00-7000-8000-000000000031", Book: "season-0", Times: 3,
		StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
	}
	lifetimeDay = books.Round{
		ID: "lifetime/day-1", Book: "lifetime", Times: 1,
		StartsAt: time.Date(2026, 10, 31, 22, 0, 0, 0, time.UTC),
	}
)

func subscriber(t *testing.T) (round_take_subscriber.Subscriber, *inmemory_round_store.Store) {
	t.Helper()
	shelf, store := inmemory_book_store.New(), inmemory_round_store.New()
	for _, book := range []books.Book{
		books.NewBook("season-0", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)),
		books.NewBook("lifetime", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	} {
		_, err := shelf.Open(t.Context(), book)
		require.NoError(t, err)
	}
	for _, round := range []books.Round{day, battle, lifetimeDay} {
		_, err := store.Open(t.Context(), round, books.Start{})
		require.NoError(t, err)
	}
	return round_take_subscriber.New(count_take_usecase.New(shelf, store, store)), store
}

func takes(account, country string, at time.Time, position uint64, from, to uint32) *planetv1.TilesTaken {
	tiles := make([]*planetv1.TakenTile, 0, to-from)
	for index := from; index < to; index++ {
		tiles = append(tiles, &planetv1.TakenTile{TileId: 40 + index, Index: index})
	}
	return &planetv1.TilesTaken{AccountId: account, Country: country, TakenAt: timestamppb.New(at), Position: position, Tiles: tiles}
}

func accountOf(t *testing.T, id string) books.AccountID {
	t.Helper()
	account, err := uuid.Parse(id)
	require.NoError(t, err)
	return books.AccountID(account)
}

func TestATakeDuringARoundCountsForItsFlagAndItsPlayer(t *testing.T) {
	handler, store := subscriber(t)

	err := handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{takes(ada, "fr", battle.StartsAt.Add(time.Minute), 9, 0, 1)})

	require.NoError(t, err)
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"fr": 1},
		Players:   map[books.AccountID]map[books.Country]uint64{accountOf(t, ada): {"fr": 1}},
	}, store.CountedIn(battle.ID))
}

func TestAnEnclosureToldInPartsAndToldTwiceCountsEachTileOnce(t *testing.T) {
	handler, store := subscriber(t)
	at := battle.StartsAt.Add(time.Minute)
	enclosure := []*planetv1.TilesTaken{takes(ada, "fr", at, 9, 0, 3), takes(ada, "fr", at, 9, 3, 7)}

	for range 2 {
		require.NoError(t, handler.HandleBatch(t.Context(), enclosure))
	}
	for _, part := range enclosure {
		require.NoError(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{part}), "a failed batch is handled again one event at a time")
	}

	for _, round := range []books.RoundID{day.ID, battle.ID} {
		assert.Equal(t, map[books.Country]uint64{"fr": 7}, store.CountedIn(round).Countries, "one ledger event in two parts, seven takes")
	}
}

func TestEachAccountRoundAndBookOfABatchCountsItsOwnTakes(t *testing.T) {
	handler, store := subscriber(t)

	err := handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{
		takes(ada, "fr", day.StartsAt.Add(time.Hour), 1, 0, 2),
		takes(grace, "de", battle.StartsAt.Add(time.Minute), 2, 0, 3),
		takes(ada, "fr", lifetimeDay.StartsAt, 3, 0, 1),
		takes(ada, "be", lifetimeDay.StartsAt.Add(time.Minute), 4, 0, 1),
	})

	require.NoError(t, err)
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"fr": 2},
		Players:   map[books.AccountID]map[books.Country]uint64{accountOf(t, ada): {"fr": 2}},
	}, store.CountedIn(day.ID), "the day of season 0 counts until the battle's takes, never scored")
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"de": 3, "fr": 1, "be": 1},
		Players: map[books.AccountID]map[books.Country]uint64{
			accountOf(t, grace): {"de": 3},
			accountOf(t, ada):   {"fr": 1, "be": 1},
		},
	}, store.CountedIn(battle.ID))
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"fr": 1, "be": 1},
		Players:   map[books.AccountID]map[books.Country]uint64{accountOf(t, ada): {"fr": 1, "be": 1}},
	}, store.CountedIn(lifetimeDay.ID), "another book's round")
}

func TestABatchWithAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler, _ := subscriber(t)
	at := battle.StartsAt
	good := takes(ada, "fr", at, 1, 0, 1)

	require.Error(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, takes("", "fr", at, 2, 0, 1)}))
	require.Error(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, takes(ada, "", at, 2, 0, 1)}))
	noTime := takes(ada, "fr", at, 2, 0, 1)
	noTime.TakenAt = nil
	require.Error(t, handler.HandleBatch(t.Context(), []*planetv1.TilesTaken{good, noTime}))
}
