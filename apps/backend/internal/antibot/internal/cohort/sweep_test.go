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

	w := New(Config{ChainWindow: 10 * time.Minute}, clock, nil)

	w.Watch(detect.Click{Scope: "2a00:8c40:f0c5:6713::/64", Tile: 1, Country: "bg", At: clock.Now()})
	require.Len(t, w.members, 1)
	require.Len(t, w.starts, 1)
	require.Len(t, w.prefixes, 1)

	clock.Advance(time.Hour)
	w.sweep()

	assert.Empty(t, w.members)
	assert.Empty(t, w.starts)
	assert.Empty(t, w.prefixes)
}

func TestSweepKeepsAScopeForTheWholeChain(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	// A track window shorter than the chain would forget its first links.
	w := New(Config{ChainWindow: 30 * time.Minute, TrackWindow: time.Minute}, clock, nil)

	w.Watch(detect.Click{Scope: "198.51.100.7", Tile: 1, Country: "bg", At: clock.Now()})

	clock.Advance(20 * time.Minute)
	w.sweep()

	assert.Len(t, w.members, 1)
}

func TestSweepReportsScopesInStep(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	var reported []int
	w := New(Config{}, clock, func(scopes int) { reported = append(reported, scopes) })

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

func TestWiden(t *testing.T) {
	for scope, want := range map[string]string{
		"2a00:8c40:f0c5:6713::/64": "2a00:8c40:f0c0::/44",
		"2a00:8c40:f0ce:fda7::/64": "2a00:8c40:f0c0::/44",
		"203.0.113.7":              "203.0.113.0/24",
		"caller":                   "",
		"10.0.0.0/8":               "",
	} {
		assert.Equal(t, want, widen(scope, 24, 44), scope)
	}
}
