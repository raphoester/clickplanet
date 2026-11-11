package close_round_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type stubTerritory struct {
	held map[books.Country]uint32
	err  error
}

func (s *stubTerritory) Snapshot(context.Context) (books.Snapshot, error) {
	tiles := uint32(0)
	for _, held := range s.held {
		tiles += held
	}
	return books.Snapshot{Tiles: tiles, Held: s.held}, s.err
}

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

var day1 = books.Round{ID: "season-1/day-1", Book: "season-1", Times: 1, Ranking: books.GainRanking{}, StartsAt: utc(8, 0)}

type stack struct {
	useCase   *close_round_usecase.UseCase
	rounds    *inmemory_round_store.Store
	territory *stubTerritory
}

func newStack(t *testing.T, open map[books.Round]books.Start) stack {
	t.Helper()
	rounds, territory := inmemory_round_store.New(), &stubTerritory{}
	for round, start := range open {
		_, err := rounds.Open(t.Context(), round, start)
		require.NoError(t, err)
	}
	return stack{useCase: close_round_usecase.New(cpatomic.FakeTransactioner{}, territory, rounds), rounds: rounds, territory: territory}
}

func (s stack) close(t *testing.T, round books.RoundID, at time.Time) books.Ending {
	t.Helper()
	ending, err := s.useCase.Execute(t.Context(), close_round_usecase.Command{Round: round, EndsAt: at})
	require.NoError(t, err)
	return ending
}

func TestARoundNeverOpenedIsKeptMissedAndEachCloseAnswersTheFirstEnd(t *testing.T) {
	stack := newStack(t, nil)

	assert.Equal(t, books.Ending{At: utc(8, 23), Missed: true}, stack.close(t, battle, utc(8, 23)))
	assert.Equal(t, books.Ending{At: utc(8, 23), Missed: true}, stack.close(t, battle, utc(9, 23)))

	missed, found, err := stack.rounds.Missed(t.Context(), battle)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, books.NewMissedRound(battle, utc(8, 23)), missed)
	_, found, err = stack.rounds.Find(t.Context(), battle)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestARoundScoredIsAnsweredAgainWithTheEndItWasScoredAt(t *testing.T) {
	stack := newStack(t, map[books.Round]books.Start{day1: {}})
	stack.territory.held = map[books.Country]uint32{"fr": 3}

	assert.Equal(t, books.Ending{At: utc(8, 21)}, stack.close(t, day1.ID, utc(8, 21)))
	stack.territory.held = map[books.Country]uint32{"de": 3}
	assert.Equal(t, books.Ending{At: utc(8, 21)}, stack.close(t, day1.ID, utc(8, 22)))

	scored, err := stack.rounds.ScoredIn(t.Context(), "season-1")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Equal(t, utc(8, 21), scored[0].Round.EndsAt)
	assert.Equal(t, []books.Result{{Country: "fr", Rank: 1, Points: 25, Share: 1}}, scored[0].Results,
		"a close told again scores nothing again")
}

func TestARoundWhoseMapCannotBeReadIsScoredByTheNextClose(t *testing.T) {
	stack := newStack(t, map[books.Round]books.Start{day1: {}})
	down := errors.New("the planet module is down")
	stack.territory.err = down

	_, err := stack.useCase.Execute(t.Context(), close_round_usecase.Command{Round: day1.ID, EndsAt: utc(8, 21)})
	require.ErrorIs(t, err, down)
	round, _, err := stack.rounds.Find(t.Context(), day1.ID)
	require.NoError(t, err)
	assert.False(t, round.Scored)

	stack.territory.err, stack.territory.held = nil, map[books.Country]uint32{"fr": 1}
	assert.Equal(t, books.Ending{At: utc(8, 21)}, stack.close(t, day1.ID, utc(8, 21)))
}

func TestADayRankedOnTheGroundHeldAndGainedIsScoredOnBothRanks(t *testing.T) {
	day2 := books.Round{ID: "season-1/day-2", Book: "season-1", Times: 1, Ranking: books.HeldAndGainRanking{}, StartsAt: utc(8, 21)}
	stack := newStack(t, map[books.Round]books.Start{day2: {"ie": 50, "br": 30, "fr": 15, "lv": 5}})
	stack.territory.held = map[books.Country]uint32{"ie": 50, "br": 28, "fr": 10, "lv": 12}

	stack.close(t, day2.ID, utc(9, 21))

	scored, err := stack.rounds.ScoredIn(t.Context(), "season-1")
	require.NoError(t, err)
	require.Len(t, scored, 1)
	assert.Equal(t, []books.Result{
		{Country: "ie", Rank: 1, Points: 25, Share: 0.5},
		{Country: "lv", Rank: 2, Points: 18, Share: 0.12},
		{Country: "br", Rank: 3, Points: 15, Share: 0.28},
		{Country: "fr", Rank: 4, Points: 12, Share: 0.1},
	}, scored[0].Results, "ie held the most and kept it, lv gained the most, fr lost ground and still scores")
}
