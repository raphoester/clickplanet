package defender_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type harness struct {
	watchdog *defender.Watchdog
	clock    *cptime.FixedClock
	owner    map[uint32]string
	next     uint32
}

func newHarness(config defender.Config) *harness {
	h := &harness{
		clock: cptime.NewFixedClock(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC)),
		owner: map[uint32]string{},
		next:  1000,
	}
	h.watchdog = defender.New(config, h.clock, nil)
	return h
}

func (h *harness) deliver(scope string, tile uint32, country string, accepted bool) detect.Verdict {
	held := h.owner[tile]

	c := detect.Click{Scope: scope, Tile: tile, Country: country, At: h.clock.Now(), Held: held, NoOp: held == country}

	verdict, _ := h.watchdog.Watch(c)

	if accepted {
		h.watchdog.Committed(c)
		if !c.NoOp {
			h.owner[tile] = country
		}
	}

	return verdict
}

func (h *harness) click(scope string, tile uint32, country string) detect.Verdict {
	return h.deliver(scope, tile, country, true)
}

func (h *harness) fresh() uint32 {
	h.next++
	return h.next
}

func config() defender.Config {
	return defender.Config{
		RetakeWindow:  time.Minute,
		MinClicks:     20,
		MinShare:      0.8,
		CertainClicks: 60,
		CertainShare:  0.95,
		TrackWindow:   10 * time.Minute,
	}
}

// queue has an attacker take n tiles from BG at once, and the suspect take them back one refill at a time.
func (h *harness) queue(suspect string, n int, refill time.Duration) detect.Verdict {
	tiles := make([]uint32, 0, n)
	for range n {
		tile := h.fresh()
		h.owner[tile] = "BG"
		h.click("attacker", tile, "FR")
		tiles = append(tiles, tile)
	}

	var verdict detect.Verdict
	for _, tile := range tiles {
		h.clock.Advance(refill)
		verdict = h.click(suspect, tile, "BG")
	}

	return verdict
}

func TestAQueueOfRetakesIsCaughtHoweverSlowItDrains(t *testing.T) {
	h := newHarness(config())

	// Retaken 1.3s to 39s after the loss: the retaker's reflex window sees almost none of it.
	var verdict detect.Verdict
	for range 3 {
		verdict = h.queue("bot", 30, 1300*time.Millisecond)
	}

	assert.Equal(t, detect.Certain, verdict)
}

func TestADefenderWhoAlsoPaintsIsClear(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 10 {
		h.queue("player", 3, 2*time.Second)

		for range 7 {
			h.clock.Advance(2 * time.Second)
			verdict = h.click("player", h.fresh(), "BG")
		}
	}

	assert.Equal(t, detect.Clear, verdict, "three in ten is a person holding a border")
}

func TestAMostlyRetakingCallerIsOnlySuspectUntilItHasEnoughClicks(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 5 {
		h.queue("keen", 5, 2*time.Second)
		h.clock.Advance(2 * time.Second)
		verdict = h.click("keen", h.fresh(), "BG")
	}

	assert.Equal(t, detect.Suspect, verdict, "30 takes, 25 of them retakes")
}

// Pinned so nobody sets a share without measuring first: two people fighting over one tile retake every click.
func TestTwoPlayersFightingOverOneTileReadAsRetakes(t *testing.T) {
	h := newHarness(config())

	const tile = uint32(7)
	h.owner[tile] = "BG"

	var verdict detect.Verdict
	for range 60 {
		h.clock.Advance(1500 * time.Millisecond)
		h.click("french", tile, "FR")
		h.clock.Advance(1500 * time.Millisecond)
		verdict = h.click("bulgarian", tile, "BG")
	}

	assert.Equal(t, detect.Certain, verdict, "why the shipped config sets no share")
}

func TestTakingATileBackForAnotherCountryIsNotARetake(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 3 {
		tiles := make([]uint32, 0, 30)
		for range 30 {
			tile := h.fresh()
			h.owner[tile] = "BG"
			h.click("attacker", tile, "FR")
			tiles = append(tiles, tile)
		}
		for _, tile := range tiles {
			h.clock.Advance(time.Second)
			verdict = h.click("rival", tile, "RO")
		}
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestTakingBackWhatYouYourselfTookIsNotARetake(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 90 {
		tile := h.fresh()
		h.owner[tile] = "BG"
		h.click("flipper", tile, "FR")
		h.clock.Advance(time.Second)
		verdict = h.click("flipper", tile, "BG")
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestALossOlderThanTheWindowIsNotARetake(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 3 {
		verdict = h.queue("late", 30, 3*time.Minute)
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestARefusedClickTakesNothingFromAnyone(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 90 {
		tile := h.fresh()
		h.owner[tile] = "BG"
		h.deliver("griefer", tile, "FR", false)
		h.clock.Advance(time.Second)
		verdict = h.click("player", tile, "BG")
	}

	assert.Equal(t, detect.Clear, verdict, "the tile never left BG")
}
