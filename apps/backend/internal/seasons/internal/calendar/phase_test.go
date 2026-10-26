package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

func TestEverySeasonHasItsDaysThenItsFinaleAndTheLastOneIsOverForever(t *testing.T) {
	seasonZeroFinale := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonOneFinale := time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC)

	assert.Equal(t, []calendar.Phase{
		{Season: 0, Stage: calendar.StageDays, EndsAt: seasonZeroFinale},
		{Season: 0, Stage: calendar.StageFinale, StartsAt: seasonZeroFinale, EndsAt: seasonZeroEnds},
		{Season: 1, Stage: calendar.StageDays, StartsAt: seasonZeroEnds, EndsAt: seasonOneFinale},
		{Season: 1, Stage: calendar.StageFinale, StartsAt: seasonOneFinale, EndsAt: seasonOneEnds},
		{Season: 1, Stage: calendar.StageOver, StartsAt: seasonOneEnds},
	}, calendar.New(twoSeasons()).Phases())
}

func TestAnEmptyCalendarHasNoPhase(t *testing.T) {
	assert.Empty(t, calendar.New(calendar.Config{}).Phases())
}
