package moments_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

func TestASeasonWhoseFinaleIsADayEndsWithNoFinaleBattle(t *testing.T) {
	end := time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)
	calendar := season.NewCalendar(season.Snapshot{Number: 0, Length: end.Sub(season.Origin)}.Restore())
	plan := moments.NewPlan(calendar, season.Cutoff(21*time.Hour), []moments.RoundID{"season-0/day-13"})

	ended, err := plan.BookClosed("season-0", end)

	require.NoError(t, err)
	assert.Equal(t, moments.SeasonEnded{Season: 0, Book: "season-0", Finale: "season-0/day-13", EndedAt: end}, ended)
}
