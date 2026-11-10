package player_leaving_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures/inmemory_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/departures/usecases/decide_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/player_leaving_subscriber"
)

var (
	ada  = standings.AccountID{15: 1}
	noon = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
)

func TestTheOtherPlayersNoLongerSeeALeavingPlayer(t *testing.T) {
	store := inmemory_departure_store.New()
	require.NoError(t, store.Change(t.Context(), ada, func(d departures.Departure) departures.Departure {
		return d.Decided(departures.Returns(noon.Add(-time.Hour)))
	}))

	require.NoError(t, player_leaving_subscriber.New(decide_departure_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerLeaving{AccountId: ada.String(), DecidedAt: timestamppb.New(noon)}))

	departure, err := store.Departure(t.Context(), ada)
	require.NoError(t, err)
	assert.True(t, departure.TakeSnapshot().Leaving)
}

func TestADecisionWithNoTimeIsRefused(t *testing.T) {
	err := player_leaving_subscriber.New(decide_departure_usecase.New(inmemory_departure_store.New())).Handle(t.Context(),
		&playerv1.PlayerLeaving{AccountId: ada.String()})

	assert.ErrorIs(t, err, departures.ErrNoTime)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_leaving_subscriber.New(decide_departure_usecase.New(inmemory_departure_store.New())).Handle(t.Context(),
		&playerv1.PlayerLeaving{DecidedAt: timestamppb.New(noon)})

	assert.ErrorIs(t, err, standings.ErrInvalidAccount)
}
