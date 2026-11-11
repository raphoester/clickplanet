package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeOffers struct {
	mu       sync.Mutex
	attended []bonuses.Entrant
	left     int
	events   chan bonuses.Event
}

func (f *fakeOffers) Attend(entrant bonuses.Entrant) (<-chan bonuses.Event, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.attended = append(f.attended, entrant)
	return f.events, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.left++
	}
}

type sink struct {
	events chan listen_for_events_usecase.Event
}

func (s sink) Send(event listen_for_events_usecase.Event) error {
	s.events <- event
	return nil
}

func listening(
	t *testing.T, caller bonuses.Caller,
) (*fakeOffers, *cprules.InForce[*bonusesv1.Rules], sink, context.CancelFunc, chan error) {
	t.Helper()

	offers := &fakeOffers{events: make(chan bonuses.Event, 4)}
	inForce := cprules.NewInForce[*bonusesv1.Rules](cptime.SystemClock{})
	frames := sink{events: make(chan listen_for_events_usecase.Event, 8)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(offers, inForce, time.Hour).Execute(ctx, caller, frames)
	}()
	return offers, inForce, frames, cancel, done
}

func next(t *testing.T, frames sink) listen_for_events_usecase.Event {
	t.Helper()

	select {
	case event := <-frames.events:
		return event
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the stream sent nothing")
		return listen_for_events_usecase.Event{}
	}
}

func TestAStreamSaysItIsLiveThenSendsWhatIsOfferedToItsCaller(t *testing.T) {
	offers, _, frames, cancel, done := listening(t, bonuses.Caller{Scope: "1.2.3.4", Account: "acc", Linked: true})
	defer cancel()

	assert.True(t, next(t, frames).Heartbeat, "a heartbeat at once")

	offers.events <- bonuses.Event{Offer: &bonuses.Offer{Token: "a-box"}}
	assert.Equal(t, "a-box", next(t, frames).Offer.Token)
	offers.events <- bonuses.Event{Taken: &bonuses.Taken{CountryID: "fr"}}
	assert.Equal(t, "fr", next(t, frames).Taken.CountryID)

	cancel()
	require.NoError(t, <-done)
	assert.Equal(t, []bonuses.Entrant{"account:acc"}, offers.attended, "a signed-in account attends as itself")
	assert.Equal(t, 1, offers.left, "a stream that ends leaves")
}

func TestAGuestsStreamAttendsAsItsAddress(t *testing.T) {
	offers, _, frames, cancel, done := listening(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"})
	next(t, frames)
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, []bonuses.Entrant{"1.2.3.4"}, offers.attended)
}

func TestEachSetTakenIsTold(t *testing.T) {
	_, inForce, frames, cancel, done := listening(t, bonuses.Caller{Scope: "1.2.3.4"})
	defer cancel()
	next(t, frames)

	set, err := rules.Seeded().Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	inForce.Enforce(set)

	assert.True(t, next(t, frames).RulesChanged)
	cancel()
	require.NoError(t, <-done)
}

type failingSink struct{}

func (failingSink) Send(listen_for_events_usecase.Event) error { return errors.New("gone") }

func TestAStreamThatCannotBeWrittenEnds(t *testing.T) {
	offers := &fakeOffers{events: make(chan bonuses.Event)}

	err := listen_for_events_usecase.New(offers, cprules.NewInForce[*bonusesv1.Rules](cptime.SystemClock{}), time.Hour).
		Execute(t.Context(), bonuses.Caller{Scope: "1.2.3.4"}, failingSink{})

	require.Error(t, err)
	assert.Equal(t, 1, offers.left)
}
