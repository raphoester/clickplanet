package shadowban

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubClock struct{ now time.Time }

func (c *stubClock) Now() time.Time { return c.now }

type stubOwner struct{}

func (stubOwner) Owner(uint32) (string, bool) { return "", false }

func observe(d *Detector, scope string, tile uint32, country string) {
	if drop, takes := d.Observe(scope, tile, country); !drop && takes {
		d.Took(scope, tile)
	}
}

func TestSweepForgetsIdleCallersAndStaleTiles(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}

	d := New(Config{
		ReactionWindow: time.Second,
		TrackWindow:    time.Minute,
		MinReactions:   4,
	}, stubOwner{}, clock, nil, nil)

	observe(d, "player", 1, "FR")
	clock.now = clock.now.Add(80 * time.Millisecond)
	observe(d, "bot", 1, "PS")

	require.Len(t, d.callers, 2)
	require.Len(t, d.tiles, 1)

	clock.now = clock.now.Add(2 * time.Hour)
	d.sweep()

	assert.Empty(t, d.callers, "callers idle past the window are dropped")
	assert.Empty(t, d.tiles, "tiles nobody can still be reacting to are dropped")
}

func TestSweepKeepsACallerServingABan(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}

	d := New(Config{
		Enforce:        true,
		ReactionWindow: time.Second,
		TrackWindow:    time.Minute,
		BanDuration:    24 * time.Hour,
		MinReactions:   4,
		MaxMedian:      250 * time.Millisecond,
		MaxSpread:      120 * time.Millisecond,
	}, stubOwner{}, clock, nil, nil)

	for i := range uint32(6) {
		observe(d, "player", 100+i, "FR")
		clock.now = clock.now.Add(80 * time.Millisecond)
		observe(d, "bot", 100+i, "PS")
		clock.now = clock.now.Add(time.Second)
	}

	require.True(t, d.Banned("bot"))

	clock.now = clock.now.Add(2 * time.Hour)
	d.sweep()

	assert.True(t, d.Banned("bot"), "a sweep must not release a ban still running")
}

func TestTheCountryTallyIsCapped(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}

	d := New(Config{ReactionWindow: time.Second, TrackWindow: time.Minute}, stubOwner{}, clock, nil, nil)

	for i := range 100 {
		d.Observe("spreader", uint32(i), string(rune('A'+i%26))+string(rune('A'+i/26)))
	}

	assert.LessOrEqual(t, len(d.callers["spreader"].countries), maxTrackedCountries)
}
