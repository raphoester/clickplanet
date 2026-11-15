package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/listen_for_events_usecase"
)

type fakeBoxes struct {
	mu       sync.Mutex
	attended []boxes.Entrant
	left     int
	offers   chan boxes.Offer
}

func (f *fakeBoxes) Attend(entrant boxes.Entrant) (<-chan boxes.Offer, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.attended = append(f.attended, entrant)
	return f.offers, func() {
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

func listening(t *testing.T, caller boxes.Caller) (*fakeBoxes, sink, context.CancelFunc, chan error) {
	t.Helper()

	offered := &fakeBoxes{offers: make(chan boxes.Offer, 4)}
	frames := sink{events: make(chan listen_for_events_usecase.Event, 8)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(offered, time.Hour).Execute(ctx, caller, frames)
	}()
	return offered, frames, cancel, done
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
	offered, frames, cancel, done := listening(t, boxes.Caller{Scope: "1.2.3.4", Account: "acc", Linked: true})
	defer cancel()

	assert.True(t, next(t, frames).Heartbeat, "a heartbeat at once")

	offered.offers <- boxes.Offer{Token: "a-box"}
	assert.Equal(t, "a-box", next(t, frames).Offer.Token)

	cancel()
	require.NoError(t, <-done)
	assert.Equal(t, []boxes.Entrant{"account:acc"}, offered.attended, "a signed-in account attends as itself")
	assert.Equal(t, 1, offered.left, "a stream that ends leaves")
}

func TestAGuestsStreamAttendsAsItsAddress(t *testing.T) {
	offered, frames, cancel, done := listening(t, boxes.Caller{Scope: "1.2.3.4", Account: "a-guest"})
	next(t, frames)
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, []boxes.Entrant{"1.2.3.4"}, offered.attended)
}

type failingSink struct{}

func (failingSink) Send(listen_for_events_usecase.Event) error { return errors.New("gone") }

func TestAStreamThatCannotBeWrittenEnds(t *testing.T) {
	offered := &fakeBoxes{offers: make(chan boxes.Offer)}

	err := listen_for_events_usecase.New(offered, time.Hour).
		Execute(t.Context(), boxes.Caller{Scope: "1.2.3.4"}, failingSink{})

	require.Error(t, err)
	assert.Equal(t, 1, offered.left)
}
