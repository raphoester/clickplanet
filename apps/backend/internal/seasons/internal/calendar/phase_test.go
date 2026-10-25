package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var zeroAndOne = calendar.New(calendar.Config{List: []calendar.Entry{
	{Number: 0, EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
	{Number: 1, EndsAt: time.Date(2026, 12, 12, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
}})

func phaseAt(t *testing.T, c calendar.Calendar, now time.Time) calendar.Phase {
	t.Helper()

	phase, ok := c.PhaseAt(now)
	assert.True(t, ok)
	return phase
}

func TestASeasonPlaysItsDaysUntilItsFinaleStarts(t *testing.T) {
	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageDays},
		phaseAt(t, zeroAndOne, time.Date(2026, 10, 31, 20, 59, 59, 0, time.UTC)))
}

func TestTheFinaleRunsFromItsStartToTheSeasonsEnd(t *testing.T) {
	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageFinale},
		phaseAt(t, zeroAndOne, time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)))
	assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageFinale},
		phaseAt(t, zeroAndOne, time.Date(2026, 10, 31, 22, 59, 59, 0, time.UTC)))
}

func TestTheNextSeasonStartsWhenOneEnds(t *testing.T) {
	assert.Equal(t, calendar.Phase{Season: 1, Stage: calendar.StageDays},
		phaseAt(t, zeroAndOne, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)))
}

func TestPastTheLastSeasonItIsOver(t *testing.T) {
	assert.Equal(t, calendar.Phase{Season: 1, Stage: calendar.StageOver},
		phaseAt(t, zeroAndOne, time.Date(2026, 12, 12, 23, 0, 0, 0, time.UTC)))
}

func TestACalendarWithNoSeasonHasNoPhase(t *testing.T) {
	_, ok := calendar.New(calendar.Config{}).PhaseAt(time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC))

	assert.False(t, ok)
}

func TestBetweenTwoSeasonsTheOneThatEndedIsOver(t *testing.T) {
	withAGap := calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
		{Number: 1, StartsAt: time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 12, 5, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
	}})

	for _, at := range []time.Time{time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 18, 59, 59, 0, time.UTC)} {
		assert.Equal(t, calendar.Phase{Season: 0, Stage: calendar.StageOver}, phaseAt(t, withAGap, at), at)
	}
	assert.Equal(t, calendar.Phase{Season: 1, Stage: calendar.StageDays},
		phaseAt(t, withAGap, time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC)))
}

func TestBeforeTheFirstSeasonStartsThereIsNoPhase(t *testing.T) {
	later := calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, StartsAt: time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
	}})

	_, ok := later.PhaseAt(time.Date(2026, 10, 31, 18, 59, 59, 0, time.UTC))

	assert.False(t, ok)
}
