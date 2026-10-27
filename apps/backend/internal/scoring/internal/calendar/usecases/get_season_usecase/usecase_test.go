package get_season_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: seasonZeroEnds})
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
