package churner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsSilentAccountsAndTheirIndexes(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 30, 13, 47, 0, 0, time.UTC))

	w := newWatchdog(bounded(), clock)
	w.Watch(detect.Click{Scope: "2001:db8:a:1::/64", Account: "guest", Tile: 1, Country: "pl", At: clock.Now()})

	clock.Advance(time.Hour)
	w.sweep()
	assert.Len(t, w.accounts, 1, "kept for the window and one handoff")

	clock.Advance(2 * time.Minute)
	w.sweep()
	assert.Empty(t, w.accounts)
	assert.Empty(t, w.born)
	assert.Empty(t, w.prefixes)
}

type sample struct {
	accounts int
	family   string
}

func TestSweepReportsTheScopesAndRelaysClickedOnSinceTheLast(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 30, 13, 47, 0, 0, time.UTC))

	var (
		churns []sample
		links  []int
	)
	w := New(bounded(), clock,
		func(accounts int, family string) { churns = append(churns, sample{accounts, family}) },
		func(n int) { links = append(links, n) },
	)

	churn(w, clock, 3)
	w.Watch(detect.Click{Scope: "203.0.113.7", Account: "player", Tile: 1, Country: "fr", At: clock.Now()})
	w.sweep()

	assert.ElementsMatch(t, []sample{
		{3, "v6"}, {1, "v6"}, {1, "v6"}, {1, "v6"}, {1, "v4"},
	}, churns, "the home line, the three lines of the relay, and the player")
	assert.ElementsMatch(t, []int{2, 2}, links, "the dz relay, and the pl accounts handing off on the home line")

	churns, links = nil, nil
	clock.Advance(2 * time.Minute)
	w.sweep()

	assert.Empty(t, churns, "nobody clicked since")
	assert.Empty(t, links)
}
