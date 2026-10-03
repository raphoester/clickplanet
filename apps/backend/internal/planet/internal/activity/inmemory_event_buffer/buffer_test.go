package inmemory_event_buffer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer"
)

var start = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func click(tile uint32) activity.Event {
	return activity.Event{
		At: start.Add(time.Duration(tile) * time.Millisecond), Kind: activity.KindClick,
		Caller: activity.Caller{Scope: "2001:db8::/64"}, Tile: tile, Country: "bg", Outcome: activity.OutcomeAccepted,
	}
}

func TestAFlushWritesWhatWasRecordedInOrderAndOnlyOnce(t *testing.T) {
	persistence := inmemory_event_buffer.NewMemoryPersistence()
	buffer := inmemory_event_buffer.New(10, persistence)

	buffer.Record(click(1))
	buffer.Record(click(2))

	flushed, err := buffer.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, inmemory_event_buffer.Flushed{Written: 2}, flushed)

	flushed, err = buffer.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, inmemory_event_buffer.Flushed{}, flushed)
	assert.Equal(t, []activity.Event{click(1), click(2)}, persistence.Saved())
	assert.Equal(t, 1, persistence.Saves(), "an empty flush writes nothing")
}

func TestARecordedEventIsTrimmed(t *testing.T) {
	persistence := inmemory_event_buffer.NewMemoryPersistence()
	buffer := inmemory_event_buffer.New(10, persistence)

	event := click(1)
	event.Country = "b\x00g"
	buffer.Record(event)

	_, err := buffer.Flush(t.Context())
	require.NoError(t, err)
	require.Len(t, persistence.Saved(), 1)
	assert.Equal(t, "bg", persistence.Saved()[0].Country)
}

func TestAFailedFlushKeepsTheEventsAheadOfTheNewOnes(t *testing.T) {
	persistence := inmemory_event_buffer.NewMemoryPersistence()
	buffer := inmemory_event_buffer.New(10, persistence)
	cause := errors.New("postgres is down")

	buffer.Record(click(1))
	persistence.Break(cause)
	_, err := buffer.Flush(t.Context())
	require.ErrorIs(t, err, cause)

	buffer.Record(click(2))
	persistence.Heal()
	flushed, err := buffer.Flush(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, flushed.Written)
	assert.Equal(t, []activity.Event{click(1), click(2)}, persistence.Saved())
}

func TestAFullBufferDropsTheNewestAndSaysHowMany(t *testing.T) {
	persistence := inmemory_event_buffer.NewMemoryPersistence()
	buffer := inmemory_event_buffer.New(2, persistence)

	for tile := range uint32(5) {
		buffer.Record(click(tile + 1))
	}

	flushed, err := buffer.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, inmemory_event_buffer.Flushed{Written: 2, Dropped: 3}, flushed)
	assert.Equal(t, []activity.Event{click(1), click(2)}, persistence.Saved())

	flushed, err = buffer.Flush(t.Context())
	require.NoError(t, err)
	assert.Zero(t, flushed.Dropped, "each drop is told once")
}

func TestAFailedFlushStaysWithinTheBound(t *testing.T) {
	persistence := inmemory_event_buffer.NewMemoryPersistence()
	buffer := inmemory_event_buffer.New(2, persistence)
	persistence.Break(errors.New("postgres is down"))

	buffer.Record(click(1))
	buffer.Record(click(2))
	_, err := buffer.Flush(t.Context())
	require.Error(t, err)

	buffer.Record(click(3))
	persistence.Heal()
	flushed, err := buffer.Flush(t.Context())

	require.NoError(t, err)
	assert.Equal(t, inmemory_event_buffer.Flushed{Written: 2, Dropped: 1}, flushed)
	assert.Equal(t, []activity.Event{click(1), click(2)}, persistence.Saved(), "the oldest are kept")
}

// flush is read while it runs: its context is cancelled once it returns.
type flush struct {
	err         error
	hasDeadline bool
}

type countingFlusher struct {
	flushes chan flush
}

func (f countingFlusher) Flush(ctx context.Context) (inmemory_event_buffer.Flushed, error) {
	_, hasDeadline := ctx.Deadline()
	f.flushes <- flush{err: ctx.Err(), hasDeadline: hasDeadline}
	return inmemory_event_buffer.Flushed{}, nil
}

func TestTheRunnerFlushesEveryIntervalAndOnceMoreWhenStopped(t *testing.T) {
	flusher := countingFlusher{flushes: make(chan flush, 100)}
	runner := inmemory_event_buffer.NewRunner(10*time.Millisecond, flusher)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		runner.Run(ctx)
		close(done)
	}()

	first := <-flusher.flushes
	assert.True(t, first.hasDeadline, "every flush has a timeout")
	cancel()
	<-done

	require.NotEmpty(t, flusher.flushes, "a flush follows the stop")
	var last flush
	for len(flusher.flushes) > 0 {
		last = <-flusher.flushes
	}
	require.NoError(t, last.err, "the last flush is not handed the cancelled context")
	assert.True(t, last.hasDeadline)
}
