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

var (
	seasonZeroEnds  = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneStarts = time.Date(2026, 11, 1, 19, 0, 0, 0, time.UTC)
)

func seasonZeroThenOne() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
		{Number: 1, StartsAt: seasonOneStarts, EndsAt: time.Date(2026, 12, 5, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
	}})
}

func TestTheSeasonIsTheOneInPlayOnTheClockAndTheNextOneToStart(t *testing.T) {
	clock := cptime.NewFixedClock(seasonZeroEnds.Add(-time.Second))
	useCase := get_season_usecase.New(seasonZeroThenOne(), clock)

	current, next := useCase.Execute(t.Context())
	require.NotNil(t, current)
	assert.Equal(t, calendar.Number(0), current.Number)
	require.NotNil(t, next)
	assert.Equal(t, seasonOneStarts, next.StartsAt)

	clock.Advance(2 * time.Second)
	current, next = useCase.Execute(t.Context())
	assert.Nil(t, current, "between two seasons none is in play")
	require.NotNil(t, next)
	assert.Equal(t, calendar.Number(1), next.Number)

	clock.Advance(seasonOneStarts.Sub(clock.Now()))
	current, next = useCase.Execute(t.Context())
	require.NotNil(t, current)
	assert.Equal(t, calendar.Number(1), current.Number)
	assert.Nil(t, next)
}
