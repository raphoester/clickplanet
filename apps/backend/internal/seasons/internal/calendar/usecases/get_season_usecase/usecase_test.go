package get_season_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
	}})
}

func TestTheSeasonIsTheOneCurrentOnTheClock(t *testing.T) {
	clock := cptime.NewFixedClock(seasonZeroEnds.Add(-time.Second))
	useCase := get_season_usecase.New(seasonZero(), clock)

	season, ok := useCase.Execute(t.Context())
	require.True(t, ok)
	assert.Equal(t, seasonZeroEnds, season.EndsAt)

	clock.Advance(2 * time.Second)
	_, ok = useCase.Execute(t.Context())
	assert.False(t, ok)
}
