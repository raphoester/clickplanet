package cohort

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsSilentScopesAndTheirIndexes(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	chain := bounds(func(r *Rules) { r.ChainWindow = 10 * time.Minute })
	w := New(Config{}, chain, clock, nil)
	empty, err := New(Config{}, chain, clock, nil).Save()
	require.NoError(t, err)

	w.Watch(detect.Click{Scope: "2a00:8c40:f0c5:6713::/64", Tile: 1, Country: "bg", At: clock.Now()})
	watched, err := w.Save()
	require.NoError(t, err)
	require.NotEqual(t, empty, watched)

	clock.Advance(time.Hour)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	assert.Equal(t, empty, swept)
}

func TestSweepKeepsAScopeForTheWholeChain(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock, nil)

	w.Watch(detect.Click{Scope: "198.51.100.7", Tile: 1, Country: "bg", At: clock.Now()})

	clock.Advance(20 * time.Minute)
	kept, err := w.Save()
	require.NoError(t, err)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	assert.Equal(t, kept, swept)
}

func TestSweepReportsScopesInStep(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	var reported []int
	w := New(Config{}, bounds(asIs), clock, func(scopes int) { reported = append(reported, scopes) })

	for i := range 40 {
		for _, scope := range []string{"198.51.100.1", "198.51.100.2"} {
			w.Watch(detect.Click{Scope: scope, Tile: uint32(i), Country: "bg", At: clock.Now()})
		}
		clock.Advance(2 * time.Second)
	}

	w.sweep()
	clock.Advance(2 * time.Minute)
	w.sweep()

	assert.Equal(t, []int{2, 0}, reported, "in step while clicking, and nobody once both went quiet")
}
