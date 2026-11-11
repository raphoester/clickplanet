package close_book_usecase_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

func closeBook(t *testing.T, useCase *close_book_usecase.UseCase, command close_book_usecase.Command) books.Ending {
	t.Helper()
	ending, err := useCase.Execute(t.Context(), command)
	require.NoError(t, err)
	return ending
}

func TestABookNeverOpenedIsKeptMissedAndEachCloseAnswersTheFirstEnd(t *testing.T) {
	shelf := inmemory_book_store.New()
	useCase := close_book_usecase.New(cpatomic.FakeTransactioner{}, shelf)

	assert.Equal(t, books.Ending{At: utc(8, 0), Missed: true}, closeBook(t, useCase, close_book_usecase.Command{Book: "season-0", EndsAt: utc(8, 0)}))
	assert.Equal(t, books.Ending{At: utc(8, 0), Missed: true}, closeBook(t, useCase, close_book_usecase.Command{Book: "season-0", EndsAt: utc(9, 0)}))

	_, found, err := shelf.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.False(t, found)
	missed, found, err := shelf.Missed(t.Context(), "season-0")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, books.NewMissedBook("season-0", utc(8, 0)), missed)
}

func TestABookClosedIsAnsweredAgainWithTheEndAndTiebreakItClosedAt(t *testing.T) {
	shelf := inmemory_book_store.New()
	_, err := shelf.Open(t.Context(), books.NewBook("season-1", utc(8, 0)))
	require.NoError(t, err)
	useCase := close_book_usecase.New(cpatomic.FakeTransactioner{}, shelf)

	assert.Equal(t, books.Ending{At: utc(8, 21)}, closeBook(t, useCase, close_book_usecase.Command{Book: "season-1", EndsAt: utc(8, 21), Tiebreak: battle}))
	assert.Equal(t, books.Ending{At: utc(8, 21)}, closeBook(t, useCase, close_book_usecase.Command{Book: "season-1", EndsAt: utc(8, 22)}))

	book, _, err := shelf.Find(t.Context(), "season-1")
	require.NoError(t, err)
	assert.Equal(t, books.NewBook("season-1", utc(8, 0)).ClosedAt(utc(8, 21), battle), book)
}

type stubTerritory struct {
	held map[books.Country]uint32
}

func (s *stubTerritory) Snapshot(context.Context) (books.Snapshot, error) {
	tiles := uint32(0)
	for _, held := range s.held {
		tiles += held
	}
	return books.Snapshot{Tiles: tiles, Held: s.held}, nil
}

func TestTheLastDayThenTheFinalBattleThenTheSeasonAreScoredOnceInTheirOrder(t *testing.T) {
	shelf, rounds, territory := inmemory_book_store.New(), inmemory_round_store.New(), &stubTerritory{}
	tx, logger := cpatomic.FakeTransactioner{}, slog.New(slog.DiscardHandler)
	openBook := log_command.New("open_book", open_book_usecase.New(tx, shelf), logger)
	openRound := log_command.New("open_round", open_round_usecase.New(tx, shelf, rounds, territory), logger)
	closeRound := log_command.New("close_round", close_round_usecase.New(tx, territory, rounds), logger)
	closeSeason := log_command.New("close_book", close_book_usecase.New(tx, shelf), logger)
	play := func() []books.Ending {
		var endings []books.Ending
		_, err := openBook.Execute(t.Context(), open_book_usecase.Command{Book: "season-0", StartsAt: utc(8, 0)})
		require.NoError(t, err)
		_, err = openRound.Execute(t.Context(), open_round_usecase.Command{
			Round: "season-0/day-24", Book: "season-0", Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(30, 21),
		})
		require.NoError(t, err)
		territory.held = map[books.Country]uint32{"fr": 3, "de": 1}
		ending, err := closeRound.Execute(t.Context(), close_round_usecase.Command{Round: "season-0/day-24", EndsAt: utc(31, 21)})
		require.NoError(t, err)
		endings = append(endings, ending)
		_, err = openRound.Execute(t.Context(), open_round_usecase.Command{
			Round: battle, Book: "season-0", Times: 3, Ranking: books.GainRanking{}, StartsAt: utc(31, 21),
		})
		require.NoError(t, err)
		territory.held = map[books.Country]uint32{"de": 4, "fr": 1}
		ending, err = closeRound.Execute(t.Context(), close_round_usecase.Command{Round: battle, EndsAt: utc(31, 23)})
		require.NoError(t, err)
		endings = append(endings, ending)
		ending, err = closeSeason.Execute(t.Context(), close_book_usecase.Command{Book: "season-0", EndsAt: utc(31, 23), Tiebreak: battle})
		require.NoError(t, err)
		return append(endings, ending)
	}

	territory.held = map[books.Country]uint32{"it": 5}
	want := []books.Ending{{At: utc(31, 21)}, {At: utc(31, 23)}, {At: utc(31, 23)}}
	assert.Equal(t, want, play())

	book, _, err := shelf.Find(t.Context(), "season-0")
	require.NoError(t, err)
	scored, err := rounds.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)
	require.Len(t, scored, 2)
	assert.Equal(t, []books.Result{{Country: "fr", Rank: 1, Points: 25, Share: 0.75}, {Country: "de", Rank: 2, Points: 18, Share: 0.25}}, scored[0].Results)
	assert.Equal(t, []books.Result{{Country: "de", Rank: 1, Points: 75, Share: 0.8}, {Country: "fr", Rank: 2, Share: 0.2}}, scored[1].Results)
	assert.Equal(t, []books.Placed{
		{Score: books.Score{Country: "de", Points: 93, RoundsWon: 1, TiebreakPoints: 75}, Rank: 1},
		{Score: books.Score{Country: "fr", Points: 25, RoundsWon: 1}, Rank: 2},
	}, book.Table(scored, nil))

	territory.held = map[books.Country]uint32{"it": 9}
	assert.Equal(t, want, play(), "each call told again answers the same")
	again, _, err := shelf.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, book, again)
	scoredAgain, err := rounds.ScoredIn(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, scored, scoredAgain, "and changes nothing")
}
