package decide_departure_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/inmemory_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/usecases/decide_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	start   = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	account = players.AccountID{15: 1}
)

func TestARequestHidesTheAccountUntilALaterRestore(t *testing.T) {
	store := inmemory_departure_store.New()
	useCase := decide_departure_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{})

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))
	left, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.True(t, left.TakeSnapshot().Leaving)

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Returns(start.Add(time.Hour))))
	back, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, back.TakeSnapshot().Leaving)
}

func TestARequestHeardAfterALaterRestoreLeavesTheAccountShown(t *testing.T) {
	store := inmemory_departure_store.New()
	useCase := decide_departure_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{})

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Returns(start.Add(time.Hour))))
	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))

	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, departure.TakeSnapshot().Leaving)
}

func TestADecisionWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_departure_store.New()

	err := decide_departure_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}).Execute(t.Context(), account, departures.Leaves(time.Time{}))

	require.ErrorIs(t, err, departures.ErrNoTime)
	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, departure.TakeSnapshot().Leaving)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_departure_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, decide_departure_usecase.New(cpatomic.FakeTransactioner{}, store, &cpatomic.FakeOutbox{}).Execute(t.Context(), account, departures.Leaves(start)))
}

func TestEachDecisionIsToldToTheOtherModules(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	useCase := decide_departure_usecase.New(cpatomic.FakeTransactioner{}, inmemory_departure_store.New(), outbox)

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Returns(start.Add(time.Hour))))
	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))

	require.Len(t, outbox.Recorded(), 2)
	assert.True(t, proto.Equal(&playerv1.PlayerReturned{AccountId: account.String(), DecidedAt: timestamppb.New(start.Add(time.Hour))},
		outbox.Recorded()[0]))
	assert.True(t, proto.Equal(&playerv1.PlayerLeaving{AccountId: account.String(), DecidedAt: timestamppb.New(start)},
		outbox.Recorded()[1]), "an older decision is told too: a listener keeps the latest")
}

func TestADecisionWithNoTimeIsNotTold(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	err := decide_departure_usecase.New(cpatomic.FakeTransactioner{}, inmemory_departure_store.New(), outbox).
		Execute(t.Context(), account, departures.Leaves(time.Time{}))

	require.ErrorIs(t, err, departures.ErrNoTime)
	assert.Empty(t, outbox.Recorded())
}
