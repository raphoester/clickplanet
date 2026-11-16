package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inprocess_level_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/inprocess_title_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/usecases/listen_for_events_usecase"
)

var (
	ada = accounts.AccountID{15: 1}

	fifth = levels.Curve{}.StandingOf(900)
)

type recordingSink struct {
	mu         sync.Mutex
	earned     []titles.ID
	reached    []levels.Standing
	heartbeats int
	err        error
}

func (r *recordingSink) SendHeartbeat() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.heartbeats++
	return nil
}

func (r *recordingSink) beats() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.heartbeats
}

func (r *recordingSink) SendTitleEarned(standing titles.Standing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.earned = append(r.earned, standing.Title().ID())
	return r.err
}

func (r *recordingSink) SendLevelReached(standing levels.Standing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reached = append(r.reached, standing)
	return r.err
}

func (r *recordingSink) received() []titles.ID {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.earned)
}

func (r *recordingSink) levels() []levels.Standing {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.reached)
}

type feeds struct {
	titles *inprocess_title_feed.Feed
	levels *inprocess_level_feed.Feed
}

func listening(t *testing.T, ctx context.Context, heartbeat time.Duration, sink *recordingSink) (feeds, <-chan error) {
	t.Helper()

	f := feeds{titles: inprocess_title_feed.New(), levels: inprocess_level_feed.New()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(f.titles, titles.NewCatalog(), f.levels, heartbeat).Execute(ctx, ada, sink)
	}()
	require.Eventually(t, func() bool { return sink.beats() > 0 }, time.Second, time.Millisecond)
	return f, done
}

func TestOfTheTitlesGrantedAtOnceOnlyTheHighestOfEachTrackIsSent(t *testing.T) {
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	f, done := listening(t, ctx, time.Hour, sink)

	f.titles.Publish(ada, titles.IDs{"settler", "raider", "retired", "og", "loyal"})
	require.Eventually(t, func() bool { return len(sink.received()) > 0 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, []titles.ID{"og", "raider", "loyal"}, sink.received())
}

func TestALevelReachedIsSentAsTheRaiseToldIt(t *testing.T) {
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	f, done := listening(t, ctx, time.Hour, sink)

	f.levels.Publish(ada, fifth)
	f.levels.Publish(accounts.AccountID{15: 2}, levels.Curve{}.StandingOf(10_206))
	require.Eventually(t, func() bool { return len(sink.levels()) > 0 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, []levels.Standing{fifth}, sink.levels())
	assert.Empty(t, sink.received())
}

func TestASinkFailureEndsTheListening(t *testing.T) {
	failed := errors.New("the stream is gone")
	for _, c := range []struct {
		name    string
		publish func(f feeds)
	}{
		{name: "on a title", publish: func(f feeds) { f.titles.Publish(ada, titles.IDs{"settler"}) }},
		{name: "on a level", publish: func(f feeds) { f.levels.Publish(ada, fifth) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, done := listening(t, t.Context(), time.Hour, &recordingSink{err: failed})

			c.publish(f)

			select {
			case err := <-done:
				require.ErrorIs(t, err, failed)
			case <-time.After(time.Second):
				require.Fail(t, "the listening did not end")
			}
		})
	}
}

func TestTheStreamSaysItIsLiveAtOnceThenAtEachBeat(t *testing.T) {
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	_, done := listening(t, ctx, 20*time.Millisecond, sink)

	require.Eventually(t, func() bool { return sink.beats() >= 3 }, time.Second, 5*time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Empty(t, sink.received())
	assert.Empty(t, sink.levels())
}
