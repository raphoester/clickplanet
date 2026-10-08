package tile_taken_fronts_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/inmemory_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/tile_taken_fronts_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = timestamppb.New(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC))

func TestATakeCountsForItsFlagAndAgainstTheCountryThatHeldTheTile(t *testing.T) {
	store := inmemory_front_store.New()

	err := tile_taken_fronts_subscriber.New(record_take_usecase.New(store)).Handle(t.Context(),
		&planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr", PreviousCountry: "de", TakenAt: at})

	require.NoError(t, err)
	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	assert.Equal(t, fronts.TallyOf(map[fronts.Country]uint64{"fr": 1}, map[fronts.Country]uint64{"de": 1}), store.Tally(id))
}

func TestAnEventWithNoAccountOrNoCountryIsRefused(t *testing.T) {
	subscriber := tile_taken_fronts_subscriber.New(record_take_usecase.New(inmemory_front_store.New()))

	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, Country: "fr", TakenAt: at}),
		players.ErrInvalidAccount)
	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: at}),
		fronts.ErrNoCountry)
}
