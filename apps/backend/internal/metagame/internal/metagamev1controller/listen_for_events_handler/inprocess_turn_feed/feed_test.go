package inprocess_turn_feed_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/listen_for_events_handler/inprocess_turn_feed"
)

type reader struct {
	mu    sync.Mutex
	turn  *metagamev1.Turn
	reads int
	err   error
}

func (r *reader) Turn(context.Context) (*metagamev1.Turn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads++
	if r.err != nil {
		return nil, r.err
	}
	return proto.CloneOf(r.turn), nil
}

func (r *reader) play(kind metagamev1.EntryKind) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.turn = &metagamev1.Turn{Kind: kind}
}

func (r *reader) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.err = err
}

func (r *reader) readCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reads
}

const (
	plain  = metagamev1.EntryKind_ENTRY_KIND_PLAIN
	battle = metagamev1.EntryKind_ENTRY_KIND_BATTLE
)

func newFeed() (*inprocess_turn_feed.Feed, *reader) {
	reader := &reader{turn: &metagamev1.Turn{Kind: plain}}
	return inprocess_turn_feed.New(reader), reader
}

func next(t *testing.T, turns <-chan *metagamev1.Turn) metagamev1.EntryKind {
	t.Helper()
	select {
	case turn := <-turns:
		return turn.GetKind()
	default:
		require.Fail(t, "no turn was sent")
		return metagamev1.EntryKind_ENTRY_KIND_UNSPECIFIED
	}
}

func quiet(t *testing.T, turns <-chan *metagamev1.Turn) {
	t.Helper()
	select {
	case turn := <-turns:
		assert.Fail(t, "a turn was sent", "%v", turn)
	default:
	}
}

func closed(t *testing.T, turns <-chan *metagamev1.Turn) {
	t.Helper()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-turns:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}

func TestAStreamGetsTheTurnOnceItIsRead(t *testing.T) {
	feed, reader := newFeed()
	reader.play(battle)

	turns := feed.Subscribe(t.Context())
	quiet(t, turns)

	feed.Refresh(t.Context())

	assert.Equal(t, battle, next(t, turns))
}

func TestAnotherStreamGetsTheTurnAtOnceWithNoRead(t *testing.T) {
	feed, reader := newFeed()
	reader.play(battle)
	first := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, first)

	second := feed.Subscribe(t.Context())

	assert.Equal(t, battle, next(t, second))
	assert.Equal(t, 1, reader.readCount())
}

func TestEachChangeOfTurnIsSentToEveryStream(t *testing.T) {
	feed, reader := newFeed()
	first := feed.Subscribe(t.Context())
	second := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, first)
	next(t, second)

	reader.play(battle)
	feed.Refresh(t.Context())

	assert.Equal(t, battle, next(t, first), "the battle starts")
	assert.Equal(t, battle, next(t, second))
}

func TestATurnThatHoldsIsNotSentAgain(t *testing.T) {
	feed, reader := newFeed()
	turns := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, turns)

	feed.Refresh(t.Context())

	assert.Equal(t, 2, reader.readCount())
	quiet(t, turns)
}

func TestATurnNobodyFollowsIsNotRead(t *testing.T) {
	feed, reader := newFeed()
	ctx, cancel := context.WithCancel(t.Context())
	turns := feed.Subscribe(ctx)
	feed.Refresh(t.Context())
	next(t, turns)

	cancel()
	closed(t, turns)

	feed.Refresh(t.Context())
	assert.Equal(t, 1, reader.readCount())
}

func TestTheFirstStreamAfterNobodyFollowedWaitsForAFreshRead(t *testing.T) {
	feed, reader := newFeed()
	ctx, cancel := context.WithCancel(t.Context())
	gone := feed.Subscribe(ctx)
	feed.Refresh(t.Context())
	next(t, gone)
	cancel()
	closed(t, gone)

	reader.play(battle)
	turns := feed.Subscribe(t.Context())
	quiet(t, turns)
	feed.Refresh(t.Context())

	assert.Equal(t, battle, next(t, turns))
}

func TestAFailedReadSendsNothingAndTheNextReadDoes(t *testing.T) {
	feed, reader := newFeed()
	reader.fail(assert.AnError)
	turns := feed.Subscribe(t.Context())

	feed.Refresh(t.Context())
	quiet(t, turns)

	reader.fail(nil)
	feed.Refresh(t.Context())

	assert.Equal(t, plain, next(t, turns))
}

func TestTheRunnerReadsForANewStreamAndStopsWithItsContext(t *testing.T) {
	feed, reader := newFeed()
	reader.play(battle)
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		feed.Run(ctx)
	}()

	turns := feed.Subscribe(t.Context())

	require.Eventually(t, func() bool {
		select {
		case turn := <-turns:
			return assert.Equal(t, battle, turn.GetKind())
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)

	cancel()
	require.Eventually(t, func() bool {
		select {
		case <-stopped:
			return true
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}
