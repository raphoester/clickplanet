package restored_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/inmemory_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/usecases/decide_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/restored_subscriber"
)

func TestARestoredAccountIsShownAgain(t *testing.T) {
	account := players.AccountID{15: 1}
	start := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	store := inmemory_departure_store.New()
	useCase := decide_departure_usecase.New(store)
	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))

	err := restored_subscriber.New(useCase).Handle(t.Context(), &authv1.AccountRestored{
		AccountId: account.String(), RestoredAt: timestamppb.New(start.Add(time.Hour)),
	})

	require.NoError(t, err)
	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, departure.Leaving())
}

func TestARestoreWithNoTimeIsRefused(t *testing.T) {
	err := restored_subscriber.New(decide_departure_usecase.New(inmemory_departure_store.New())).
		Handle(t.Context(), &authv1.AccountRestored{AccountId: players.AccountID{15: 1}.String()})

	assert.ErrorIs(t, err, departures.ErrNoTime)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := restored_subscriber.New(decide_departure_usecase.New(inmemory_departure_store.New())).
		Handle(t.Context(), &authv1.AccountRestored{RestoredAt: timestamppb.Now()})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
