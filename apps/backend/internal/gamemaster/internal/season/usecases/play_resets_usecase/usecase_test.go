package play_resets_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges/inmemory_siege_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOne      = seasonZeroEnds.Add(16 * time.Hour)
)

type planet struct {
	reset []season.Number
	err   error
}

func (p *planet) Reset(_ context.Context, starts season.Window) error {
	if p.err != nil {
		return p.err
	}
	p.reset = append(p.reset, starts.Number)
	return nil
}

func harness(t *testing.T, at time.Time) (*play_resets_usecase.UseCase, *planet, *cptime.FixedClock) {
	t.Helper()
	useCase, told, clock, _ := harnessWith(t, at, season.Snapshot{Number: 0, Length: seasonZeroEnds.Sub(season.Origin)}, 16*time.Hour)
	return useCase, told, clock
}

func harnessWith(t *testing.T, at time.Time, zero season.Snapshot, pause time.Duration,
) (*play_resets_usecase.UseCase, *planet, *cptime.FixedClock, *inmemory_siege_store.Store) {
	t.Helper()
	seasons, judged := inmemory_season_store.New(), inmemory_siege_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 1, Break: pause, Length: 28 * 24 * time.Hour}.Restore()))
	told, clock := &planet{}, cptime.NewFixedClock(at)
	planner := moments.NewPlanner(seasons, inmemory_entry_store.New(), inmemory_moment_store.New(), judged, season.Cutoff(21*time.Hour))
	return play_resets_usecase.New(planner, told, clock), told, clock, judged
}

// A siege as season 0's finale, with season 1 starting as it ends: the reset waits for the map from before the siege,
// or the siege's end would put the old season's map back on the new one.
func TestASeasonThatStartsAfterASiegeResetsOnceItsArenaClosed(t *testing.T) {
	zero := season.Snapshot{
		Number: 0, Length: seasonZeroEnds.Sub(season.Origin),
		Finale: season.Finale{Length: 2 * time.Hour, Set: schedule.SetName("capital-siege"), Times: 3},
	}
	useCase, told, clock, judged := harnessWith(t, seasonZeroEnds, zero, 0)
	_, err := judged.KeepJudged(t.Context(), moments.Judgement{Round: moments.FinaleRoundOf(0), Mode: moments.Siege})
	require.NoError(t, err)

	_, sent, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.False(t, sent, "the siege's arena is open")

	_, err = judged.KeepClosed(t.Context(), moments.FinaleRoundOf(0))
	require.NoError(t, err)
	clock.Advance(3 * time.Second)
	_, sent, err = useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.True(t, sent)
	assert.Equal(t, []season.Number{1}, told.reset)
}

func TestSeasonOnesStartResetsTheMapOnce(t *testing.T) {
	useCase, told, clock := harness(t, seasonOne.Add(-time.Second))

	for range 3 {
		_, _, err := useCase.Execute(t.Context())
		require.NoError(t, err)
		clock.Advance(time.Second)
	}

	assert.Equal(t, []season.Number{1}, told.reset, "at 15:00, and not again")
}

func TestSeasonZeroAndTheBreakResetNothing(t *testing.T) {
	for _, at := range []time.Time{seasonZeroEnds.Add(-time.Hour), seasonZeroEnds, seasonOne.Add(-time.Second)} {
		useCase, told, _ := harness(t, at)

		_, told2, err := useCase.Execute(t.Context())

		require.NoError(t, err)
		assert.False(t, told2, "%s", at)
		assert.Empty(t, told.reset)
	}
}

func TestAResetRedisDidNotTakeIsSentAgain(t *testing.T) {
	useCase, told, clock := harness(t, seasonOne)
	down := errors.New("redis is down")
	told.err = down

	_, _, err := useCase.Execute(t.Context())
	require.ErrorIs(t, err, down)

	told.err = nil
	clock.Advance(time.Second)
	_, sent, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.True(t, sent)
	assert.Equal(t, []season.Number{1}, told.reset)
}
