package jury

import (
	"testing"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBanner struct{}

type fixedWatchdog struct {
	name    string
	verdict detect.Verdict
}

func (w *fixedWatchdog) Name() string { return w.name }

func (w *fixedWatchdog) Attempted(detect.Click) {}

func (w *fixedWatchdog) Watch(detect.Click) (detect.Verdict, detect.Evidence) {
	return w.verdict, detect.Evidence{}
}

func (w *fixedWatchdog) Committed(detect.Click) {}

func (stubBanner) Flag(string) (shadowban.Sentence, bool) { return shadowban.Sentence{Flags: 1}, true }

func (stubBanner) Banned(string) bool { return false }

func (stubBanner) Flagged() int { return 0 }

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	j := New(Config{TrackWindow: time.Minute}, stubBanner{}, clock, Hooks{})

	j.Inspect(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})
	require.Len(t, j.callers, 1)

	clock.Advance(2 * time.Hour)
	j.sweep()

	assert.Empty(t, j.callers, "without this the map keeps an entry for every caller that ever clicked")
}

func TestTheCountryTallyIsCapped(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	j := New(Config{}, stubBanner{}, clock, Hooks{})

	for i := range 100 {
		j.Inspect(detect.Click{
			Scope:   "spreader",
			Tile:    uint32(i),
			Country: string(rune('A'+i%26)) + string(rune('A'+i/26)),
			At:      clock.Now(),
		})
	}

	assert.LessOrEqual(t, len(j.callers["spreader"].countries), maxTrackedCountries,
		"a client must not be able to spend memory by cycling through country codes")
}

func TestOnlyTheLastFewTilesAreKept(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	j := New(Config{}, stubBanner{}, clock, Hooks{})

	for i := range uint32(100) {
		j.Inspect(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.Len(t, j.callers["caller"].tiles, keptTiles)
}

func TestTheSweepReportsWhoIsStanding(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	standing := map[string]int{}

	unsure := &fixedWatchdog{name: "unsure", verdict: detect.Suspect}
	sure := &fixedWatchdog{name: "sure", verdict: detect.Clear}

	j := New(Config{SuspicionWindow: 10 * time.Minute, TrackWindow: time.Hour}, stubBanner{}, clock, Hooks{
		OnStanding: func(watchdog string, level detect.Verdict, callers int) {
			standing[watchdog+" "+level.String()] = callers
		},
	}, unsure, sure)

	// Read long enough ago that its suspicion has lapsed, but not forgotten.
	j.Inspect(detect.Click{Scope: "stale", At: clock.Now()})
	clock.Advance(20 * time.Minute)

	j.Inspect(detect.Click{Scope: "first", At: clock.Now()})
	j.Inspect(detect.Click{Scope: "second", At: clock.Now()})

	sure.verdict = detect.Certain
	j.Inspect(detect.Click{Scope: "third", At: clock.Now()})

	j.sweep()

	assert.Equal(t, map[string]int{
		"unsure suspect": 3,
		"unsure certain": 0,
		"sure suspect":   1,
		"sure certain":   1,
	}, standing, "every watchdog and level, zero included, or a gauge never falls back")
}
