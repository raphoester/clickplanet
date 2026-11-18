package season_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

const cutoff = season.Cutoff(21 * time.Hour)

func TestASeasonThatStartsAtACutoffHasAFullFirstDay(t *testing.T) {
	start := time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC)
	window := season.Window{Number: 1, StartsAt: start, EndsAt: start.Add(2 * day)}

	assert.Equal(t, []season.Day{
		{Season: 1, Number: 1, StartsAt: start, EndsAt: start.Add(day)},
		{Season: 1, Number: 2, StartsAt: start.Add(day), EndsAt: start.Add(2 * day)},
	}, window.Days(cutoff), "a day that ends at the season's end is its last")
}

func TestADayRunsFromItsStartToJustBeforeItsEnd(t *testing.T) {
	first := planned(zero).Windows()[0].Days(cutoff)[0]

	assert.True(t, first.Runs(first.StartsAt))
	assert.True(t, first.Runs(first.EndsAt.Add(-time.Nanosecond)))
	assert.False(t, first.Runs(first.EndsAt))
}

func TestACutoffIsATimeOfDay(t *testing.T) {
	require.NoError(t, season.Cutoff(0).Check())
	require.NoError(t, cutoff.Check())
	require.Error(t, season.Cutoff(day).Check())
	require.Error(t, season.Cutoff(-time.Minute).Check())
}
