package jury

import (
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubClock struct{ now time.Time }

func (c *stubClock) Now() time.Time { return c.now }

type stubBanner struct{}

func (stubBanner) Flag(string) (int, bool) { return 1, true }

func (stubBanner) Banned(string) bool { return false }

func (stubBanner) Flagged() int { return 0 }

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	j := New(Config{TrackWindow: time.Minute}, stubBanner{}, clock, nil)

	j.Inspect(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.now})
	require.Len(t, j.callers, 1)

	clock.now = clock.now.Add(2 * time.Hour)
	j.sweep()

	assert.Empty(t, j.callers, "without this the map keeps an entry for every caller that ever clicked")
}

func TestTheCountryTallyIsCapped(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	j := New(Config{}, stubBanner{}, clock, nil)

	for i := range 100 {
		j.Inspect(detect.Click{
			Scope:   "spreader",
			Tile:    uint32(i),
			Country: string(rune('A'+i%26)) + string(rune('A'+i/26)),
			At:      clock.now,
		})
	}

	assert.LessOrEqual(t, len(j.callers["spreader"].countries), maxTrackedCountries,
		"a client must not be able to spend memory by cycling through country codes")
}

func TestOnlyTheLastFewTilesAreKept(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	j := New(Config{}, stubBanner{}, clock, nil)

	for i := range uint32(100) {
		j.Inspect(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.now})
	}

	assert.Len(t, j.callers["caller"].tiles, keptTiles)
}
