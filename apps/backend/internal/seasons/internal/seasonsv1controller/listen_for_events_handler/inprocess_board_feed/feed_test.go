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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ana  = standings.AccountID{15: 1}
	kofi = standings.AccountID{15: 2}
	lea  = standings.AccountID{15: 3}
)

type kept struct {
	board    *seasonsv1.Board
	accounts []standings.AccountID
}

type reader struct {
	mu     sync.Mutex
	boards map[string]kept
	reads  map[string]int
	err    error
}

func newReader() *reader {
	return &reader{boards: map[string]kept{}, reads: map[string]int{}}
}

func (r *reader) Board(_ context.Context, country string) (*seasonsv1.Board, []standings.AccountID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reads[country]++
	if r.err != nil {
		return nil, nil, r.err
	}
	shown, ok := r.boards[country]
	if !ok {
		return &seasonsv1.Board{}, []standings.AccountID{}, nil
	}
	return proto.CloneOf(shown.board), shown.accounts, nil
}

func (r *reader) show(country string, accounts []standings.AccountID, names ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	board := &seasonsv1.Board{}
	for _, name := range names {
		board.Standings = append(board.Standings, &seasonsv1.Standing{Name: name})
	}
	r.boards[country] = kept{board: board, accounts: accounts}
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

func TestAStreamGetsItsBoardOnceItIsRead(t *testing.T) {
	f := newFixture()
	f.reader.show("", []standings.AccountID{ana}, "Ana")

	boards := f.feed.Subscribe(t.Context(), "")
	quiet(t, boards)

	f.feed.Refresh(t.Context())

	assert.Equal(t, []string{"Ana"}, next(t, boards))
}

func TestAnotherStreamOfTheViewGetsTheBoardAtOnceWithNoRead(t *testing.T) {
	f := newFixture()
	f.reader.show("fr", []standings.AccountID{ana}, "Ana")
	first := f.feed.Subscribe(t.Context(), "fr")
	f.feed.Refresh(t.Context())
	next(t, first)

	second := f.feed.Subscribe(t.Context(), "fr")

	assert.Equal(t, []string{"Ana"}, next(t, second))
	assert.Equal(t, 1, f.reader.readsOf("fr"))
}

func TestEveryStreamOfAViewGetsTheNewBoard(t *testing.T) {
	f := newFixture()
	first := f.feed.Subscribe(t.Context(), "")
	second := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, first)
	next(t, second)

	f.reader.show("", []standings.AccountID{ana}, "Ana")
	f.feed.MarkTaken(standings.Take{Account: ana, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, []string{"Ana"}, next(t, first))
	assert.Equal(t, []string{"Ana"}, next(t, second))
}

func TestATakeMovesTheBoardOfEveryPlayerAndTheBoardOfItsFlag(t *testing.T) {
	f := newFixture()
	for _, country := range []string{"", "fr", "de"} {
		f.feed.Subscribe(t.Context(), country)
	}
	f.feed.Refresh(t.Context())

	f.feed.MarkTaken(standings.Take{Account: ana, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, 2, f.reader.readsOf(""))
	assert.Equal(t, 2, f.reader.readsOf("fr"))
	assert.Equal(t, 1, f.reader.readsOf("de"), "a take for France moves no German board that does not list its taker")
}

func TestATakeMovesNoOtherFlagsBoardEvenOneThatListsItsTaker(t *testing.T) {
	f := newFixture()
	f.reader.show("de", []standings.AccountID{kofi, ana}, "Kofi", "Ana")
	f.feed.Subscribe(t.Context(), "de")
	f.feed.Refresh(t.Context())

	f.feed.MarkTaken(standings.Take{Account: ana, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, 1, f.reader.readsOf("de"), "Ana's tiles for Germany did not change")
}

func TestAMovedBoardIsReadAtMostOnceASecond(t *testing.T) {
	f := newFixture()
	f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())

	f.feed.MarkTaken(standings.Take{Account: ana, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.Every-time.Millisecond)
	f.feed.MarkTaken(standings.Take{Account: kofi, Country: "gh"})
	f.feed.Refresh(t.Context())
	assert.Equal(t, 1, f.reader.readsOf(""))

	f.refreshAfter(t, time.Millisecond)
	assert.Equal(t, 2, f.reader.readsOf(""), "two takes in one second are one read")

	f.refreshAfter(t, inprocess_board_feed.Every)
	assert.Equal(t, 2, f.reader.readsOf(""), "nothing moved since")
}

func TestAFollowedBoardIsReadAgainEveryFifteenSecondsWithNoTake(t *testing.T) {
	f := newFixture()
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, boards)

	f.reader.show("", []standings.AccountID{ana}, "Ana renamed")
	f.refreshAfter(t, inprocess_board_feed.AtLeast-time.Millisecond)
	quiet(t, boards)

	f.refreshAfter(t, time.Millisecond)
	assert.Equal(t, []string{"Ana renamed"}, next(t, boards))
}

func TestABoardThatDidNotChangeIsNotSentAgain(t *testing.T) {
	f := newFixture()
	f.reader.show("", []standings.AccountID{ana}, "Ana")
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())
	next(t, boards)

	f.feed.MarkTaken(standings.Take{Account: kofi, Country: "gh"})
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, 2, f.reader.readsOf(""))
	quiet(t, boards)
}

func TestAStreamThatReadsNothingGetsOnlyTheNewestBoard(t *testing.T) {
	f := newFixture()
	boards := f.feed.Subscribe(t.Context(), "")
	f.feed.Refresh(t.Context())

	f.reader.show("", []standings.AccountID{ana}, "Ana")
	f.feed.MarkTaken(standings.Take{Account: ana, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.Every)
	f.reader.show("", []standings.AccountID{ana, kofi}, "Ana", "Kofi")
	f.feed.MarkTaken(standings.Take{Account: kofi, Country: "gh"})
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, []string{"Ana", "Kofi"}, next(t, boards))
	quiet(t, boards)
}

func TestAForgottenAccountMovesOnlyTheBoardsThatListIt(t *testing.T) {
	f := newFixture()
	f.reader.show("fr", []standings.AccountID{ana}, "Ana")
	f.reader.show("gh", []standings.AccountID{kofi}, "Kofi")
	f.feed.Subscribe(t.Context(), "fr")
	f.feed.Subscribe(t.Context(), "gh")
	f.feed.Refresh(t.Context())

	f.feed.MarkForgotten(ana)
	f.refreshAfter(t, inprocess_board_feed.Every)

	assert.Equal(t, 2, f.reader.readsOf("fr"))
	assert.Equal(t, 1, f.reader.readsOf("gh"))
}

func TestABoardNobodyFollowsIsNeitherReadNorMoved(t *testing.T) {
	f := newFixture()
	ctx, cancel := context.WithCancel(t.Context())
	boards := f.feed.Subscribe(ctx, "fr")
	f.feed.Refresh(t.Context())
	next(t, boards)

	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, open := <-boards:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)

	f.feed.MarkTaken(standings.Take{Account: lea, Country: "fr"})
	f.refreshAfter(t, inprocess_board_feed.AtLeast)
	assert.Equal(t, 1, f.reader.readsOf("fr"))
}

func TestAFailedReadIsTriedAgainASecondLater(t *testing.T) {
	f := newFixture()
	f.reader.show("", []standings.AccountID{ana}, "Ana")
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
	f.reader.show("", []standings.AccountID{ana}, "Ana")
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
