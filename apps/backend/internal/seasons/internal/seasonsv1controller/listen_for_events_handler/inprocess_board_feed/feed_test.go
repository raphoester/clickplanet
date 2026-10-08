package inprocess_board_feed_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler/inprocess_board_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type reader struct {
	mu     sync.Mutex
	boards map[string]*seasonsv1.Board
	reads  map[string]int
	err    error
}

func newReader() *reader {
	return &reader{boards: map[string]*seasonsv1.Board{}, reads: map[string]int{}}
}

func (r *reader) Board(_ context.Context, country string) (*seasonsv1.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads[country]++
	if r.err != nil {
		return nil, r.err
	}
	board, ok := r.boards[country]
	if !ok {
		return &seasonsv1.Board{}, nil
	}
	return proto.CloneOf(board), nil
}

func (r *reader) show(country string, names ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	board := &seasonsv1.Board{}
	for _, name := range names {
		board.Standings = append(board.Standings, &seasonsv1.Standing{Name: name})
	}
	r.boards[country] = board
}

func (r *reader) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.err = err
}

func (r *reader) readsOf(country string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.reads[country]
}

type fixture struct {
	feed   *inprocess_board_feed.Feed
	reader *reader
	clock  *cptime.FixedClock
}

func newFixture() fixture {
	reader := newReader()
	clock := cptime.NewFixedClock(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))
	return fixture{feed: inprocess_board_feed.New(reader, clock), reader: reader, clock: clock}
}

func (f fixture) refreshAfter(t *testing.T, elapsed time.Duration) {
	f.clock.Advance(elapsed)
	f.feed.Refresh(t.Context())
}

func namesOf(board *seasonsv1.Board) []string {
	names := make([]string, 0, len(board.GetStandings()))
	for _, standing := range board.GetStandings() {
		names = append(names, standing.GetName())
	}
	return names
}

func next(t *testing.T, boards <-chan *seasonsv1.Board) []string {
	t.Helper()
	select {
	case board := <-boards:
		return namesOf(board)
	default:
		require.Fail(t, "no board was sent")
		return nil
	}
}

func quiet(t *testing.T, boards <-chan *seasonsv1.Board) {
	t.Helper()
	select {
	case board := <-boards:
		assert.Fail(t, "a board was sent", "%v", namesOf(board))
	default:
	}
}

func closed(t *testing.T, boards <-chan *seasonsv1.Board) {
	t.Helper()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-boards:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}

func TestAStreamGetsItsBoardOnceItIsRead(t *testing.T) {
	f := newFixture()
	f.reader.show("", "Ana")

	boards := f.feed.Subscribe(t.Context(), "")
	quiet(t, boards)

	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"Ana"}, next(t, boards))
}

func TestAnotherStreamOfTheViewGetsTheBoardAtOnceWithNoRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr", "Ana")
	first := f.feed.Subscribe(t.Context(), "fr")
	f.feed.Refresh(t.Context())
	next(t, first)

	second := f.feed.Subscribe(t.Context(), "fr")
	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"Ana"}, next(t, second))
	assert.Equal(t, 1, f.reader.readsOf("fr"))
}

func TestOneReadServesEveryStreamOfAView(t *testing.T) {
	f := newFixture()
	f.reader.show("", "Ana")
	streams := []<-chan *seasonsv1.Board{
		f.feed.Subscribe(t.Context(), ""),
		f.feed.Subscribe(t.Context(), ""),
		f.feed.Subscribe(t.Context(), ""),
	}

	f.feed.Refresh(t.Context())

	for _, boards := range streams {
		assert.Equal(t, []string{"Ana"}, next(t, boards))
	}
	assert.Equal(t, 1, f.reader.readsOf(""))
}

func TestAFollowedBoardIsReadAgainOnItsOwnClock(t *testing.T) {
	f := newFixture()
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, boards)

	f.reader.show("", "Ana")
	f.refreshAfter(t, inprocess_board_feed.Every-time.Millisecond)
	assert.Equal(t, 1, f.reader.readsOf(""))
	quiet(t, boards)

	f.refreshAfter(t, time.Millisecond)

	assert.Equal(t, 2, f.reader.readsOf(""))
	assert.Equal(t, []string{"Ana"}, next(t, boards))
}

func TestEachViewIsReadOnItsOwnClock(t *testing.T) {
	f := newFixture()
	f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())

	f.clock.Advance(inprocess_board_feed.Every / 2)
	f.feed.Subscribe(t.Context(), "fr")
	f.feed.Refresh(t.Context())
	assert.Equal(t, 1, f.reader.readsOf(""), "a new view is no reason to read the others")
	assert.Equal(t, 1, f.reader.readsOf("fr"), "a new view is read at once")

	f.refreshAfter(t, inprocess_board_feed.Every/2)
	assert.Equal(t, 2, f.reader.readsOf(""))
	assert.Equal(t, 1, f.reader.readsOf("fr"))

	f.refreshAfter(t, inprocess_board_feed.Every/2)
	assert.Equal(t, 2, f.reader.readsOf(""))
	assert.Equal(t, 2, f.reader.readsOf("fr"))
}

func TestEveryStreamOfAViewGetsTheNewBoard(t *testing.T) {
	f := newFixture()
	first := f.feed.Subscribe(t.Context(), "")
	second := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, first)
	next(t, second)

	f.reader.show("", "Ana")
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, []string{"Ana"}, next(t, first))
	assert.Equal(t, []string{"Ana"}, next(t, second))
}

func TestABoardThatDidNotChangeIsNotSentAgain(t *testing.T) {
	f := newFixture()
	f.reader.show("", "Ana")
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, boards)

	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, 2, f.reader.readsOf(""))
	quiet(t, boards)
}

func TestAStreamThatReadsNothingGetsOnlyTheNewestBoard(t *testing.T) {
	f := newFixture()
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())

	f.reader.show("", "Ana")
	f.refreshAfter(t, inprocess_board_feed.Every)
	f.reader.show("", "Ana", "Kofi")
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, []string{"Ana", "Kofi"}, next(t, boards))
	quiet(t, boards)
}

func TestABoardNobodyFollowsIsNotRead(t *testing.T) {
	f := newFixture()
	ctx, cancel := context.WithCancel(t.Context())
	boards := f.feed.Subscribe(ctx, "fr")
	f.feed.Refresh(t.Context())
	next(t, boards)

	cancel()
	closed(t, boards)

	f.refreshAfter(t, inprocess_board_feed.Every)
	assert.Equal(t, 1, f.reader.readsOf("fr"))
}

func TestTheFirstStreamAfterNobodyFollowedWaitsForAFreshRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr", "Ana")
	ctx, cancel := context.WithCancel(t.Context())
	gone := f.feed.Subscribe(ctx, "fr")
	f.feed.Refresh(t.Context())
	next(t, gone)
	cancel()
	closed(t, gone)

	f.reader.show("fr", "Kofi")
	boards := f.feed.Subscribe(t.Context(), "fr")
	quiet(t, boards)
	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"Kofi"}, next(t, boards))
	assert.Equal(t, 2, f.reader.readsOf("fr"))
}

func TestAFailedReadIsTriedAgainOnTheNextRead(t *testing.T) {
	f := newFixture()
	f.reader.show("", "Ana")
	f.reader.fail(assert.AnError)
	boards := f.feed.Subscribe(t.Context(), "")

	f.feed.Refresh(t.Context())
	quiet(t, boards)
	f.refreshAfter(t, inprocess_board_feed.Every-time.Millisecond)
	assert.Equal(t, 1, f.reader.readsOf(""))

	f.reader.fail(nil)
	f.refreshAfter(t, time.Millisecond)

	assert.Equal(t, []string{"Ana"}, next(t, boards))
}

func TestTheRunnerReadsANewViewAndStopsWithItsContext(t *testing.T) {
	f := newFixture()
	f.reader.show("", "Ana")
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		f.feed.Run(ctx)
	}()

	boards := f.feed.Subscribe(t.Context(), "")

	require.Eventually(t, func() bool {
		select {
		case board := <-boards:
			return assert.Equal(t, []string{"Ana"}, namesOf(board))
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
