package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/listen_for_events_usecase"
)

type fakeQuizzes struct {
	mu       sync.Mutex
	attended []quizzes.Entrant
	left     int
	events   chan quizzes.Event
}

func (f *fakeQuizzes) Attend(entrant quizzes.Entrant) (<-chan quizzes.Event, func()) {
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

func listening(t *testing.T, caller quizzes.Caller) (*fakeQuizzes, sink, context.CancelFunc, chan error) {
	t.Helper()

	asked := &fakeQuizzes{events: make(chan quizzes.Event, 4)}
	frames := sink{events: make(chan listen_for_events_usecase.Event, 8)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(asked, time.Hour).Execute(ctx, caller, frames)
	}()
	return asked, frames, cancel, done
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
	asked, frames, cancel, done := listening(t, quizzes.Caller{Scope: "1.2.3.4", Account: "acc", Linked: true})
	defer cancel()

	assert.True(t, next(t, frames).Heartbeat, "a heartbeat at once")

	asked.events <- quizzes.Event{Offer: &quizzes.Offer{Token: "a-quiz"}}
	assert.Equal(t, "a-quiz", next(t, frames).Offer.Token)
	asked.events <- quizzes.Event{Won: &quizzes.Won{CountryID: "fr", Subject: "ee"}}
	assert.Equal(t, quizzes.Won{CountryID: "fr", Subject: "ee"}, *next(t, frames).Won)

	cancel()
	require.NoError(t, <-done)
	assert.Equal(t, []quizzes.Entrant{"account:acc"}, asked.attended, "a signed-in account attends as itself")
	assert.Equal(t, 1, asked.left, "a stream that ends leaves")
}

func TestAGuestsStreamAttendsAsItsAddress(t *testing.T) {
	asked, frames, cancel, done := listening(t, quizzes.Caller{Scope: "1.2.3.4", Account: "a-guest"})
	next(t, frames)
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, []quizzes.Entrant{"1.2.3.4"}, asked.attended)
}

type failingSink struct{}

func (failingSink) Send(listen_for_events_usecase.Event) error { return errors.New("gone") }

func TestAStreamThatCannotBeWrittenEnds(t *testing.T) {
	asked := &fakeQuizzes{events: make(chan quizzes.Event)}

	err := listen_for_events_usecase.New(asked, time.Hour).Execute(t.Context(), quizzes.Caller{Scope: "1.2.3.4"}, failingSink{})

	require.Error(t, err)
	assert.Equal(t, 1, asked.left)
}
