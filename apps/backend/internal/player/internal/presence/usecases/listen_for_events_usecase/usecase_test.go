package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

type frame struct {
	roster    []presence.Entry
	change    *presence.Change
	heartbeat bool
}

type recordingSink struct {
	mu     sync.Mutex
	frames []frame
	err    error
}

func (s *recordingSink) SendRoster(roster []presence.Entry) error {
	return s.add(frame{roster: roster})
}

func (s *recordingSink) SendChange(change presence.Change) error {
	return s.add(frame{change: &change})
}

func (s *recordingSink) SendHeartbeat() error {
	return s.add(frame{heartbeat: true})
}

func (s *recordingSink) add(f frame) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.frames = append(s.frames, f)
	return s.err
}

func (s *recordingSink) sent() []frame {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]frame(nil), s.frames...)
}

func TestTheRosterComesFirstThenEachChange(t *testing.T) {
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(presence.Visit{Account: players.AccountID{15: 1}, Author: players.Author{Name: "Ada_L"}, Tag: "aaaaaa", Country: "fr", At: now})
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() { done <- listen_for_events_usecase.New(visits, time.Hour).Execute(ctx, sink) }()
	require.Eventually(t, func() bool { return len(sink.sent()) == 1 }, time.Second, time.Millisecond)
	visits.Forget(players.AccountID{15: 1})
	require.Eventually(t, func() bool { return len(sink.sent()) == 2 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	frames := sink.sent()
	require.Len(t, frames[0].roster, 1)
	assert.Equal(t, "Ada_L", frames[0].roster[0].Name)
	require.NotNil(t, frames[1].change)
	assert.True(t, frames[1].change.Left)
	assert.Equal(t, frames[0].roster[0].Key, frames[1].change.Entry.Key)
}

func TestAQuietRosterSendsHeartbeats(t *testing.T) {
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		done <- listen_for_events_usecase.New(inmemory_visit_storage.New(cptime.NewFixedClock(now)), time.Millisecond).Execute(ctx, sink)
	}()
	require.Eventually(t, func() bool { return len(sink.sent()) >= 3 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Empty(t, sink.sent()[0].roster)
	assert.True(t, sink.sent()[1].heartbeat)
}

func TestASinkThatFailsEndsTheStream(t *testing.T) {
	sink := &recordingSink{err: errors.New("client gone")}

	err := listen_for_events_usecase.New(inmemory_visit_storage.New(cptime.NewFixedClock(now)), time.Hour).Execute(t.Context(), sink)

	assert.EqualError(t, err, "client gone")
}
