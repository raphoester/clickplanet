package play_resets_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
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
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 0, Length: seasonZeroEnds.Sub(season.Origin)}.Restore()))
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 1, Break: 16 * time.Hour, Length: 28 * 24 * time.Hour}.Restore()))
	told, clock := &planet{}, cptime.NewFixedClock(at)
	return play_resets_usecase.New(seasons, told, clock), told, clock
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
