package raids_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
)

var at = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)

func TestARaidModeRunsFromItsStartUntilItsEnd(t *testing.T) {
	mode, err := raids.Start(at, time.Hour, "a raid of guests")
	require.NoError(t, err)

	assert.False(t, mode.RunsAt(at.Add(-time.Second)))
	assert.True(t, mode.RunsAt(at))
	assert.True(t, mode.RunsAt(at.Add(time.Hour-time.Nanosecond)))
	assert.False(t, mode.RunsAt(at.Add(time.Hour)), "it ends by itself")
}

func TestARaidModeEndedNowRunsNoMoreAndKeepsWhenItStarted(t *testing.T) {
	mode, err := raids.Start(at, time.Hour, "")
	require.NoError(t, err)

	ended := mode.Ended(at.Add(time.Minute))

	assert.False(t, ended.RunsAt(at.Add(time.Minute)))
	assert.Equal(t, raids.Snapshot{StartedAt: at, EndsAt: at.Add(time.Minute)}, ended.TakeSnapshot())
	assert.Equal(t, ended, ended.Ended(at.Add(time.Hour)), "ended twice, the first end stays")
}

func TestARaidModeRunsTwentyFourHoursAtMost(t *testing.T) {
	_, err := raids.DurationOf(24*time.Hour + time.Second)
	require.ErrorIs(t, err, raids.ErrInvalidDuration)

	_, err = raids.Snapshot{StartedAt: at, EndsAt: at.Add(25 * time.Hour)}.Restore()
	require.ErrorIs(t, err, raids.ErrInvalidDuration, "a row past the cap is not a raid mode")

	asked, err := raids.DurationOf(24 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 24*time.Hour, asked)
}

func TestARaidModeRunsWholeSecondsAndNeverBackwards(t *testing.T) {
	for _, asked := range []time.Duration{-time.Second, 1500 * time.Millisecond} {
		_, err := raids.DurationOf(asked)
		require.ErrorIs(t, err, raids.ErrInvalidDuration, "%s", asked)
	}
	_, err := raids.Start(at, 0, "")
	require.ErrorIs(t, err, raids.ErrInvalidDuration, "no duration ends one, it starts none")
}
