package replays_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

var now = time.Date(2026, 10, 12, 12, 17, 0, 0, time.UTC)

func TestAReplayWithNoEndAskedForEndsNow(t *testing.T) {
	window, err := replays.Request{Span: 12 * time.Hour}.Window(now)

	require.NoError(t, err)
	assert.Equal(t, replays.Window{Since: time.Date(2026, 10, 12, 0, 17, 0, 0, time.UTC), Until: now}, window)
}

func TestAReplayEndsWhenItIsAskedTo(t *testing.T) {
	until := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

	window, err := replays.Request{Span: 90 * time.Minute, Until: until}.Window(now)

	require.NoError(t, err)
	assert.Equal(t, replays.Window{Since: time.Date(2026, 10, 10, 22, 30, 0, 0, time.UTC), Until: until}, window)
}

func TestAReplayThatSpansNoTimeIsRefused(t *testing.T) {
	for _, span := range []time.Duration{0, -time.Hour} {
		_, err := replays.Request{Span: span}.Window(now)

		require.ErrorIs(t, err, replays.ErrInvalidRequest)
	}
}
