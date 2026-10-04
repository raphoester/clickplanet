package seen_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
)

var now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestAMomentInThePastIsKeptAsItIs(t *testing.T) {
	at := now.Add(-time.Minute)

	until, err := seen.Until(at, now)

	require.NoError(t, err)
	assert.Equal(t, at, until)
}

func TestAMomentAheadOfTheServerIsCutToNow(t *testing.T) {
	until, err := seen.Until(now.Add(time.Hour), now)

	require.NoError(t, err)
	assert.Equal(t, now, until)
}

func TestNoTimeIsRefused(t *testing.T) {
	for _, at := range []time.Time{{}, time.UnixMilli(0), time.UnixMilli(-1)} {
		_, err := seen.Until(at, now)

		assert.ErrorIs(t, err, seen.ErrNoTime, "%s", at)
	}
}
