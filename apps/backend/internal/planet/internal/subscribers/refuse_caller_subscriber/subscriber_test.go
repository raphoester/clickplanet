package refuse_caller_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/refuse_caller_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

func refuse(scope string, until, decided time.Time) *planetv1.RefuseCaller {
	return &planetv1.RefuseCaller{
		Caller:    &planetv1.RefuseCaller_Scope{Scope: scope},
		Until:     timestamppb.New(until),
		DecidedAt: timestamppb.New(decided),
	}
}

func TestARefusalIsKeptForTheScopeOfTheAddressItNames(t *testing.T) {
	book := refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at))

	require.NoError(t, refuse_caller_subscriber.New(book).Handle(t.Context(), refuse("2001:db8:1:2:aa::1", at.Add(time.Hour), at)))

	refusal, refused, err := book.Refusal(t.Context(), clicks.Payer{Scope: "2001:db8:1:2::/64"})
	require.NoError(t, err)
	require.True(t, refused)
	assert.Equal(t, at.Add(time.Hour), refusal.Until())
}

func TestARefusalSentAgainOrOutOfOrderChangesNothing(t *testing.T) {
	store := refusals.NewMemoryStore()
	subscriber := refuse_caller_subscriber.New(refusals.NewBook(store, cptime.NewFixedClock(at)))
	extended := refuse("203.0.113.7", at.Add(24*time.Hour), at.Add(time.Minute))

	require.NoError(t, subscriber.Handle(t.Context(), extended))
	require.NoError(t, subscriber.Handle(t.Context(), refuse("203.0.113.7", at.Add(time.Hour), at)))
	require.NoError(t, subscriber.Handle(t.Context(), extended))

	assert.Equal(t, refusals.Snapshot{Until: at.Add(24 * time.Hour), DecidedAt: at.Add(time.Minute)},
		store.Kept(refusals.Caller{Scope: "203.0.113.7"}))
}

func TestACommandThatNamesNoCallerIsRefused(t *testing.T) {
	subscriber := refuse_caller_subscriber.New(refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at)))

	err := subscriber.Handle(t.Context(), &planetv1.RefuseCaller{Until: timestamppb.New(at), DecidedAt: timestamppb.New(at)})

	assert.ErrorIs(t, err, refusals.ErrNoCaller)
}
