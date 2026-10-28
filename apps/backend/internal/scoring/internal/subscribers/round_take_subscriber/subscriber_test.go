package round_take_subscriber_test

import (
	"testing"
	"time"

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

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var battle = books.Round{
	ID: "0192f5a0-7c00-7000-8000-000000000031", Book: "season-0", Times: 3,
	StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
}

func subscriber(t *testing.T) (round_take_subscriber.Subscriber, *inmemory_round_store.Store) {
	t.Helper()
	shelf, store := inmemory_book_store.New(), inmemory_round_store.New()
	_, err := shelf.Open(t.Context(), books.NewBook("season-0", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, err)
	_, err = store.Open(t.Context(), battle)
	require.NoError(t, err)
	return round_take_subscriber.New(count_take_usecase.New(shelf, store, store)), store
}

func TestATakeDuringARoundCountsForItsFlagAndItsPlayer(t *testing.T) {
	handler, store := subscriber(t)

	err := handler.Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(battle.StartsAt.Add(time.Minute)), Position: 9,
	})

	require.NoError(t, err)
	counted := store.CountedIn(battle.ID)
	assert.Equal(t, map[books.Country]uint64{"fr": 1}, counted.Countries)
	assert.Len(t, counted.Players, 1)
}

func TestAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler, _ := subscriber(t)
	at := timestamppb.New(battle.StartsAt)

	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, Country: "fr", TakenAt: at}))
	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: at}))
	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
}
