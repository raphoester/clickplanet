package seasonmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/seasonmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

func TestScheduledOfSendsTheSeasonsDays(t *testing.T) {
	window := season.Window{
		Number:   1,
		StartsAt: time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 11, 3, 23, 0, 0, 0, time.UTC),
	}

	scheduled := seasonmessage.ScheduledOf(season.Snapshot{Number: 1}, window, season.Cutoff(21*time.Hour))

	days := scheduled.GetDays()
	require.Len(t, days, 3)
	assert.Equal(t, uint32(1), days[0].GetNumber())
	assert.Equal(t, window.StartsAt, days[0].GetStartsAt().AsTime())
	assert.Equal(t, time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC), days[0].GetEndsAt().AsTime())
	assert.Equal(t, time.Date(2026, 11, 3, 21, 0, 0, 0, time.UTC), days[2].GetEndsAt().AsTime())
	assert.Equal(t, "season-1/day-3", days[2].GetRoundId())
}
