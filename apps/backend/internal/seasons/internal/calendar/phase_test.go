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
