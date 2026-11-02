package deletion_requested_subscriber_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/deletion_requested_subscriber"
)

func TestARequestedDeletionHidesTheAccount(t *testing.T) {
	account := players.AccountID{15: 1}
	store := inmemory_departure_store.New()

	err := deletion_requested_subscriber.New(decide_departure_usecase.New(store)).Handle(t.Context(), &authv1.AccountDeletionRequested{
		AccountId: account.String(), RequestedAt: timestamppb.New(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)),
	})

	require.NoError(t, err)
	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.True(t, departure.Leaving())
}

func TestARequestWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_departure_store.New()

	err := deletion_requested_subscriber.New(decide_departure_usecase.New(store)).Handle(t.Context(), &authv1.AccountDeletionRequested{
		AccountId: players.AccountID{15: 1}.String(),
	})

	assert.ErrorIs(t, err, departures.ErrNoTime)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := deletion_requested_subscriber.New(decide_departure_usecase.New(inmemory_departure_store.New())).
		Handle(t.Context(), &authv1.AccountDeletionRequested{RequestedAt: timestamppb.Now()})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
