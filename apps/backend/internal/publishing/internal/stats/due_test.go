package stats_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
)

func TestTheDaysSnapshotIsDueFrom0817UTC(t *testing.T) {
	today := time.Date(2026, 11, 10, 8, 17, 0, 0, time.UTC)

	assert.Equal(t, today, stats.DueSince(today))
	assert.Equal(t, today, stats.DueSince(today.Add(15*time.Hour)))
	assert.Equal(t, today.AddDate(0, 0, -1), stats.DueSince(today.Add(-time.Minute)), "before it, yesterday's")
	paris := time.FixedZone("Paris", 3600)
	assert.Equal(t, today, stats.DueSince(time.Date(2026, 11, 10, 9, 30, 0, 0, paris)), "whatever the clock's zone")
}
