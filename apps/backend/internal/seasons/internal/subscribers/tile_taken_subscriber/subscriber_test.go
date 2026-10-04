package tile_taken_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/tile_taken_subscriber"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_contribution_store.Store) tile_taken_subscriber.Subscriber {
	return tile_taken_subscriber.New(record_take_usecase.New(calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
	}}), store))
}

func TestATakeCountsForItsFlagAtTheTimeItWasTaken(t *testing.T) {
	store := inmemory_contribution_store.New()

	err := subscriber(store).Handle(t.Context(), &planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(seasonZeroEnds.Add(-time.Hour)),
	})

	require.NoError(t, err)
	id, err := standings.AccountIDOf(account)
	require.NoError(t, err)
	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1}}, store.Tally(0, id))
}

func TestAnEventWithNoAccountNoCountryOrNoTimeIsRefused(t *testing.T) {
	handler := subscriber(inmemory_contribution_store.New())
	at := timestamppb.New(seasonZeroEnds.Add(-time.Hour))

	require.ErrorIs(t, handler.Handle(t.Context(), &planetv1.TileTaken{TileId: 42, Country: "fr", TakenAt: at}),
		standings.ErrInvalidAccount)
	require.ErrorIs(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, TakenAt: at}),
		standings.ErrNoCountry)
	assert.Error(t, handler.Handle(t.Context(), &planetv1.TileTaken{AccountId: account, TileId: 42, Country: "fr"}))
}
