package inprocess_turn_feed_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler/inprocess_turn_feed"
)

type reader struct {
	mu    sync.Mutex
	now   *gamemasterv1.GetTurnResponse
	reads int
	err   error
}

func (r *reader) Now(context.Context) (*gamemasterv1.GetTurnResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads++
	if r.err != nil {
		return nil, r.err
	}
	return proto.CloneOf(r.now), nil
}

func (r *reader) play(kind gamemasterv1.EntryKind) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.now.Turn = &gamemasterv1.Turn{Kind: kind}
}

func (r *reader) schedule(battles ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.now.Timeline = &gamemasterv1.Timeline{}
	for _, id := range battles {
		r.now.Timeline.Battles = append(r.now.Timeline.Battles, &gamemasterv1.Battle{EntryId: id})
	}
}

func (r *reader) debrief(day uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.now.Debrief = &gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Day{Day: &gamemasterv1.DayEnded{Day: day}}}
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
	plain  = gamemasterv1.EntryKind_ENTRY_KIND_PLAIN
	battle = gamemasterv1.EntryKind_ENTRY_KIND_BATTLE
)

func newFeed() (*inprocess_turn_feed.Feed, *reader) {
	reader := &reader{now: &gamemasterv1.GetTurnResponse{Turn: &gamemasterv1.Turn{Kind: plain}, Timeline: &gamemasterv1.Timeline{}}}
	return inprocess_turn_feed.New(reader), reader
}

func next(t *testing.T, turns <-chan *gamemasterv1.GetTurnResponse) gamemasterv1.EntryKind {
	t.Helper()
	return nextNow(t, turns).GetTurn().GetKind()
}

func nextNow(t *testing.T, turns <-chan *gamemasterv1.GetTurnResponse) *gamemasterv1.GetTurnResponse {
	t.Helper()
	select {
	case now := <-turns:
		return now
	default:
		require.Fail(t, "no turn was sent")
		return nil
	}
}

func quiet(t *testing.T, turns <-chan *gamemasterv1.GetTurnResponse) {
	t.Helper()
	select {
	case turn := <-turns:
		assert.Fail(t, "a turn was sent", "%v", turn)
	default:
	}
}

func closed(t *testing.T, turns <-chan *gamemasterv1.GetTurnResponse) {
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

func TestABattleAddedEditedOrDeletedIsSentToEveryStream(t *testing.T) {
	feed, reader := newFeed()
	turns := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, turns)

	for _, battles := range [][]string{{"finale"}, {"finale", "rush"}, {"rush"}} {
		reader.schedule(battles...)
		feed.Refresh(t.Context())

		timeline := nextNow(t, turns).GetTimeline().GetBattles()
		sent := make([]string, 0, len(timeline))
		for _, battle := range timeline {
			sent = append(sent, battle.GetEntryId())
		}
		assert.Equal(t, battles, sent)
	}
}

func TestANewDebriefIsSentToEveryStream(t *testing.T) {
	feed, reader := newFeed()
	turns := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, turns)

	reader.debrief(24)
	feed.Refresh(t.Context())

	assert.Equal(t, uint32(24), nextNow(t, turns).GetDebrief().GetDay().GetDay())
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
			return assert.Equal(t, battle, turn.GetTurn().GetKind())
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
