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

func newFeed() (*inprocess_race_feed.Feed, *reader) {
	reader := &reader{race: raceOf()}
	return inprocess_race_feed.New(reader), reader
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

func closed(t *testing.T, races <-chan *seasonsv1.Race) {
	t.Helper()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-races:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}

func TestAStreamGetsTheRaceOnceItIsRead(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")

	races := feed.Subscribe(t.Context())
	quiet(t, races)

	feed.Refresh(t.Context())

	assert.Equal(t, []string{"fr"}, next(t, races))
}

func TestAnotherStreamGetsTheRaceAtOnceWithNoRead(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")
	first := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, first)

	second := feed.Subscribe(t.Context())

	assert.Equal(t, []string{"fr"}, next(t, second))
	assert.Equal(t, 1, reader.readCount())
}

func TestEachReadSendsANewRaceToEveryStream(t *testing.T) {
	feed, reader := newFeed()
	first := feed.Subscribe(t.Context())
	second := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, first)
	next(t, second)

	reader.show("de", "fr")
	feed.Refresh(t.Context())

	assert.Equal(t, []string{"de", "fr"}, next(t, first))
	assert.Equal(t, []string{"de", "fr"}, next(t, second))
}

func TestARaceThatDidNotChangeIsNotSentAgain(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")
	races := feed.Subscribe(t.Context())
	feed.Refresh(t.Context())
	next(t, races)

	feed.Refresh(t.Context())

	assert.Equal(t, 2, reader.readCount())
	quiet(t, races)
}

func TestARaceNobodyFollowsIsNotRead(t *testing.T) {
	feed, reader := newFeed()
	ctx, cancel := context.WithCancel(t.Context())
	races := feed.Subscribe(ctx)
	feed.Refresh(t.Context())
	next(t, races)

	cancel()
	closed(t, races)

	feed.Refresh(t.Context())
	assert.Equal(t, 1, reader.readCount())
}

func TestTheFirstStreamAfterNobodyFollowedWaitsForAFreshRead(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")
	ctx, cancel := context.WithCancel(t.Context())
	gone := feed.Subscribe(ctx)
	feed.Refresh(t.Context())
	next(t, gone)
	cancel()
	closed(t, gone)

	reader.show("de")
	races := feed.Subscribe(t.Context())
	quiet(t, races)
	feed.Refresh(t.Context())

	assert.Equal(t, []string{"de"}, next(t, races))
}

func TestAFailedReadSendsNothingAndTheNextReadDoes(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")
	reader.fail(assert.AnError)
	races := feed.Subscribe(t.Context())

	feed.Refresh(t.Context())
	quiet(t, races)

	reader.fail(nil)
	feed.Refresh(t.Context())

	assert.Equal(t, []string{"fr"}, next(t, races))
}

func TestTheRunnerReadsForANewStreamAndStopsWithItsContext(t *testing.T) {
	feed, reader := newFeed()
	reader.show("fr")
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		feed.Run(ctx)
	}()

	races := feed.Subscribe(t.Context())

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
