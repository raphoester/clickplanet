package get_numbered_season_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_numbered_season_usecase"
)

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func TestTheSeasonIsTheOneWithTheNumber(t *testing.T) {
	useCase := get_numbered_season_usecase.New(calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
	}}))

	season, ok := useCase.Execute(t.Context(), 0)
	require.True(t, ok)
	assert.Equal(t, seasonZeroEnds, season.EndsAt)

	_, ok = useCase.Execute(t.Context(), 1)
	assert.False(t, ok)
}
