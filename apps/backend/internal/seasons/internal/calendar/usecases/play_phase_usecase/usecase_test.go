package play_phase_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/play_phase_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

var seasonZero = calendar.New(calendar.Config{List: []calendar.Entry{
	{Number: 0, EndsAt: finaleStarts.Add(2 * time.Hour), Finale: 2 * time.Hour},
}})

type game struct {
	played []calendar.Phase
	err    error
}

func (g *game) Play(_ context.Context, phase calendar.Phase) error {
	if g.err != nil {
		return g.err
	}
	g.played = append(g.played, phase)
	return nil
}

func TestTheFirstPassPlaysThePhaseOfNow(t *testing.T) {
	g := &game{}
	clock := cptime.NewFixedClock(finaleStarts.Add(time.Hour))

	played, err := play_phase_usecase.New(seasonZero, g, clock).Execute(t.Context())

	require.NoError(t, err)
	finale := calendar.Phase{Season: 0, Stage: calendar.StageFinale}
	assert.Equal(t, finale, played)
	assert.Equal(t, []calendar.Phase{finale}, g.played, "a boot mid-finale tells the game again")
}

func TestAPhaseIsPlayedOnceAndTheNextWhenItComes(t *testing.T) {
	g := &game{}
	clock := cptime.NewFixedClock(finaleStarts.Add(-time.Minute))
	useCase := play_phase_usecase.New(seasonZero, g, clock)

	for range 3 {
		_, err := useCase.Execute(t.Context())
		require.NoError(t, err)
	}
	clock.Advance(time.Minute)
	played, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	again, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	clock.Advance(2 * time.Hour)
	over, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageFinale}, played)
	assert.Equal(t, calendar.Phase{}, again, "nothing changed")
	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageOver}, over)
	assert.Equal(t, []calendar.Phase{
		{Season: 0, Stage: calendar.StageDays},
		{Season: 0, Stage: calendar.StageFinale},
		{Season: 0, Stage: calendar.StageOver},
	}, g.played)
}

func TestAPhaseTheGameRefusedIsTriedAgain(t *testing.T) {
	g := &game{err: errors.New("planet is down")}
	clock := cptime.NewFixedClock(finaleStarts)
	useCase := play_phase_usecase.New(seasonZero, g, clock)

	_, err := useCase.Execute(t.Context())
	require.Error(t, err)

	g.err = nil
	played, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageFinale}, played)
}

func TestNoSeasonPlaysNothing(t *testing.T) {
	g := &game{}

	played, err := play_phase_usecase.New(calendar.New(calendar.Config{}), g, cptime.NewFixedClock(finaleStarts)).
		Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, calendar.Phase{}, played)
	assert.Empty(t, g.played)
}
