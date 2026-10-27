package round_take_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/count_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/round_take_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var battle = rounds.Round{
	EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: true,
	Own: true, StartsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), Times: 3,
}

func TestATakeDuringARoundOfItsOwnCountsForItsFlagAndItsPlayer(t *testing.T) {
	store := inmemory_round_store.New()
	require.NoError(t, store.Open(t.Context(), battle))

	err := round_take_subscriber.New(count_take_usecase.New(store)).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(battle.StartsAt.Add(time.Minute)), Position: 9,
	})

	require.NoError(t, err)
	counted := store.CountedIn(battle)
	assert.Equal(t, map[rounds.Country]uint64{"fr": 1}, counted.Countries)
	assert.Len(t, counted.Players, 1)
}

func TestAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler := round_take_subscriber.New(count_take_usecase.New(inmemory_round_store.New()))
	at := timestamppb.New(battle.StartsAt)

	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, Country: "fr", TakenAt: at}))
	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: at}))
	require.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
}
