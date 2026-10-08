package inprocess_race_feed_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_race_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type reader struct {
	mu    sync.Mutex
	race  *seasonsv1.Race
	reads int
	err   error
}

func (r *reader) Race(context.Context) (*seasonsv1.Race, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads++
	if r.err != nil {
		return nil, r.err
	}
	return proto.CloneOf(r.race), nil
}

func (r *reader) show(leaders ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.race = raceOf(leaders...)
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

func raceOf(leaders ...string) *seasonsv1.Race {
	race := &seasonsv1.Race{}
	for _, country := range leaders {
		race.Scores = append(race.Scores, &seasonsv1.Score{CountryId: country})
	}
	return race
}

func leadersOf(race *seasonsv1.Race) []string {
	leaders := make([]string, 0, len(race.GetScores()))
	for _, score := range race.GetScores() {
		leaders = append(leaders, score.GetCountryId())
	}
	return leaders
}

type fixture struct {
	feed   *inprocess_race_feed.Feed
	reader *reader
	clock  *cptime.FixedClock
}

func newFixture() fixture {
	reader := &reader{race: raceOf()}
	clock := cptime.NewFixedClock(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))
	return fixture{feed: inprocess_race_feed.New(reader, clock), reader: reader, clock: clock}
}

func (f fixture) refreshAfter(t *testing.T, elapsed time.Duration) {
	f.clock.Advance(elapsed)
	f.feed.Refresh(t.Context())
}

func next(t *testing.T, races <-chan *seasonsv1.Race) []string {
	t.Helper()
	select {
	case race := <-races:
		return leadersOf(race)
	default:
		require.Fail(t, "no race was sent")
		return nil
	}
}

func quiet(t *testing.T, races <-chan *seasonsv1.Race) {
	t.Helper()
	select {
	case race := <-races:
		assert.Fail(t, "a race was sent", "%v", leadersOf(race))
	default:
	}
}

func TestAStreamGetsTheRaceOnceItIsRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")

	races := f.feed.Subscribe(t.Context())
	quiet(t, races)

	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"fr"}, next(t, races))
}

func TestAnotherStreamGetsTheRaceAtOnceWithNoRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")
	first := f.feed.Subscribe(t.Context())
	f.feed.Refresh(t.Context())
	next(t, first)

	second := f.feed.Subscribe(t.Context())

	assert.Equal(t, []string{"fr"}, next(t, second))
	assert.Equal(t, 1, f.reader.readCount())
}

func TestACensusMovesTheRaceForEveryStream(t *testing.T) {
	f := newFixture()
	first := f.feed.Subscribe(t.Context())
	second := f.feed.Subscribe(t.Context())
	f.feed.Refresh(t.Context())
	next(t, first)
	next(t, second)

	f.reader.show("de", "fr")
	f.feed.MarkCounted()
	f.refreshAfter(t, inprocess_race_feed.Every)

	assert.Equal(t, []string{"de", "fr"}, next(t, first))
	assert.Equal(t, []string{"de", "fr"}, next(t, second))
}

func TestAMovedRaceIsReadAtMostOnceASecond(t *testing.T) {
	f := newFixture()
	f.feed.Subscribe(t.Context())
	f.feed.Refresh(t.Context())

	f.feed.MarkCounted()
	f.refreshAfter(t, inprocess_race_feed.Every-time.Millisecond)
	assert.Equal(t, 1, f.reader.readCount())

	f.refreshAfter(t, time.Millisecond)
	assert.Equal(t, 2, f.reader.readCount())

	f.refreshAfter(t, inprocess_race_feed.Every)
	assert.Equal(t, 2, f.reader.readCount(), "nothing was counted since")
}

func TestAFollowedRaceIsReadAgainEveryMinuteWithNoCensus(t *testing.T) {
	f := newFixture()
	races := f.feed.Subscribe(t.Context())
	f.feed.Refresh(t.Context())
	next(t, races)

	f.reader.show("it")
	f.refreshAfter(t, inprocess_race_feed.AtLeast-time.Millisecond)
	quiet(t, races)

	f.refreshAfter(t, time.Millisecond)
	assert.Equal(t, []string{"it"}, next(t, races))
}

func TestARaceThatDidNotChangeIsNotSentAgain(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")
	races := f.feed.Subscribe(t.Context())
	f.feed.Refresh(t.Context())
	next(t, races)

	f.feed.MarkCounted()
	f.refreshAfter(t, inprocess_race_feed.Every)

	assert.Equal(t, 2, f.reader.readCount())
	quiet(t, races)
}

func TestARaceNobodyFollowsIsNotRead(t *testing.T) {
	f := newFixture()
	ctx, cancel := context.WithCancel(t.Context())
	races := f.feed.Subscribe(ctx)
	f.feed.Refresh(t.Context())
	next(t, races)

	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-races:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)

	f.feed.MarkCounted()
	f.refreshAfter(t, inprocess_race_feed.AtLeast)
	assert.Equal(t, 1, f.reader.readCount())
}

func TestTheFirstStreamAfterNobodyFollowedWaitsForAFreshRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")
	ctx, cancel := context.WithCancel(t.Context())
	gone := f.feed.Subscribe(ctx)
	f.feed.Refresh(t.Context())
	next(t, gone)
	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-gone:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)

	f.reader.show("de")
	races := f.feed.Subscribe(t.Context())
	quiet(t, races)
	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"de"}, next(t, races))
}

func TestAFailedReadIsTriedAgainASecondLater(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")
	f.reader.fail(assert.AnError)
	races := f.feed.Subscribe(t.Context())

	f.feed.Refresh(t.Context())
	quiet(t, races)
	f.refreshAfter(t, inprocess_race_feed.Every-time.Millisecond)
	assert.Equal(t, 1, f.reader.readCount())

	f.reader.fail(nil)
	f.refreshAfter(t, time.Millisecond)

	assert.Equal(t, []string{"fr"}, next(t, races))
}

func TestTheRunnerReadsForANewStreamAndStopsWithItsContext(t *testing.T) {
	f := newFixture()
	f.reader.show("fr")
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		f.feed.Run(ctx)
	}()

	races := f.feed.Subscribe(t.Context())

	require.Eventually(t, func() bool {
		select {
		case race := <-races:
			return assert.Equal(t, []string{"fr"}, leadersOf(race))
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
