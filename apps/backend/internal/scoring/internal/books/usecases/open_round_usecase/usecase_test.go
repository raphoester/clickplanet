package open_round_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type stubTerritory struct {
	held map[books.Country]uint32
	err  error
}

func (s *stubTerritory) Snapshot(context.Context) (books.Snapshot, error) {
	return books.Snapshot{Held: s.held}, s.err
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

type stack struct {
	useCase   *open_round_usecase.UseCase
	books     *inmemory_book_store.Store
	rounds    *inmemory_round_store.Store
	territory *stubTerritory
}

func newStack(t *testing.T, open ...books.Book) stack {
	t.Helper()
	shelf, rounds, territory := inmemory_book_store.New(), inmemory_round_store.New(), &stubTerritory{}
	for _, book := range open {
		_, err := shelf.Open(t.Context(), book)
		require.NoError(t, err)
	}
	return stack{
		useCase: open_round_usecase.New(cpatomic.FakeTransactioner{}, shelf, rounds, territory),
		books:   shelf, rounds: rounds, territory: territory,
	}
}

func command(round books.RoundID, times uint32, at time.Time) open_round_usecase.Command {
	return open_round_usecase.Command{Round: round, Book: "season-1", Times: times, Ranking: books.GainRanking{}, StartsAt: at}
}

func (s stack) found(t *testing.T, round books.RoundID) bool {
	t.Helper()
	_, found, err := s.rounds.Find(t.Context(), round)
	require.NoError(t, err)
	return found
}

func TestARoundOfABookNeverOpenedOrClosedIsRefused(t *testing.T) {
	never := newStack(t)
	_, err := never.useCase.Execute(t.Context(), command("season-1/day-1", 1, utc(8, 0)))
	require.ErrorIs(t, err, books.ErrRefused)
	require.ErrorIs(t, err, open_round_usecase.ErrBookNotOpen)
	assert.False(t, never.found(t, "season-1/day-1"))

	closed := newStack(t, books.NewBook("season-1", utc(8, 0)).ClosedAt(utc(31, 23), ""))
	_, err = closed.useCase.Execute(t.Context(), command(battle, 3, utc(31, 23)))
	require.ErrorIs(t, err, open_round_usecase.ErrBookNotOpen)
	assert.False(t, closed.found(t, battle))
}

func TestARoundACloseMissedNeverOpens(t *testing.T) {
	stack := newStack(t, books.NewBook("season-1", utc(8, 0)))
	require.NoError(t, stack.rounds.KeepMissed(t.Context(), books.NewMissedRound(battle, utc(8, 23))))

	_, err := stack.useCase.Execute(t.Context(), command(battle, 2, utc(8, 21)))

	require.ErrorIs(t, err, books.ErrRefused)
	require.ErrorIs(t, err, books.ErrMissed)
	assert.False(t, stack.found(t, battle))
}

func TestARoundOfOneToTenTimesOpensOnceAndKeepsItsFirstRanking(t *testing.T) {
	stack := newStack(t, books.NewBook("season-1", utc(8, 0)))
	for _, times := range []uint32{0, books.MaxTimes + 1} {
		_, err := stack.useCase.Execute(t.Context(), command("season-1/day-1", times, utc(8, 0)))
		require.ErrorIs(t, err, books.ErrInvalidTimes)
	}

	opened, err := stack.useCase.Execute(t.Context(), command("season-1/day-1", 1, utc(8, 0)))
	require.NoError(t, err)
	assert.True(t, opened)
	again := command("season-1/day-1", 2, utc(9, 0))
	again.Ranking = books.HeldAndGainRanking{}
	opened, err = stack.useCase.Execute(t.Context(), again)
	require.NoError(t, err)
	assert.False(t, opened)

	round, _, err := stack.rounds.Find(t.Context(), "season-1/day-1")
	require.NoError(t, err)
	assert.Equal(t, books.Round{ID: "season-1/day-1", Book: "season-1", Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(8, 0)}, round)
}

func TestARoundOpenedWhileTheMapCannotBeReadOpensOnTheNextCall(t *testing.T) {
	stack := newStack(t, books.NewBook("season-1", utc(8, 0)))
	_, err := stack.useCase.Execute(t.Context(), command("season-1/day-1", 1, utc(8, 0)))
	require.NoError(t, err)
	stack.territory.held, stack.territory.err = map[books.Country]uint32{"fr": 3}, errors.New("planet is down")

	_, err = stack.useCase.Execute(t.Context(), command(battle, 2, utc(8, 20)))
	require.Error(t, err)
	assert.False(t, stack.found(t, battle))

	stack.territory.err = nil
	_, err = stack.useCase.Execute(t.Context(), command(battle, 2, utc(8, 20)))
	require.NoError(t, err)
	start, err := stack.rounds.StartOf(t.Context(), battle)
	require.NoError(t, err)
	assert.Equal(t, books.Start{"fr": 3}, start)
}

func TestARoundToldToCountFromNothingOrTheFirstOfItsBookReadsNoMap(t *testing.T) {
	stack := newStack(t, books.NewBook("season-1", utc(8, 0)))
	stack.territory.held, stack.territory.err = map[books.Country]uint32{"fr": 3}, errors.New("planet is down")

	_, err := stack.useCase.Execute(t.Context(), command("season-1/day-1", 1, utc(8, 0)))
	require.NoError(t, err, "the book's first round")
	fromNothing := command(battle, 2, utc(8, 21))
	fromNothing.FromNothing = true
	_, err = stack.useCase.Execute(t.Context(), fromNothing)
	require.NoError(t, err)

	for _, round := range []books.RoundID{"season-1/day-1", battle} {
		start, err := stack.rounds.StartOf(t.Context(), round)
		require.NoError(t, err)
		assert.Empty(t, start, round)
	}
}
