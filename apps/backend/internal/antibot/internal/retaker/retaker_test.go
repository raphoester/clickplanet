package retaker_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type harness struct {
	watchdog  *retaker.Watchdog
	clock     *cptime.FixedClock
	owner     map[uint32]string
	reactions []time.Duration
	ground    map[uint32]string
}

func newHarness(config retaker.Config) *harness {
	h := &harness{
		clock:  cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)),
		owner:  map[uint32]string{},
		ground: map[uint32]string{},
	}

	h.watchdog = retaker.New(config, h.clock, func(d time.Duration) {
		h.reactions = append(h.reactions, d)
	})

	return h
}

func (h *harness) click(scope string, tile uint32, country string) detect.Verdict {
	return h.deliver(scope, tile, country, true)
}

func (h *harness) refused(scope string, tile uint32, country string) detect.Verdict {
	return h.deliver(scope, tile, country, false)
}

func (h *harness) deliver(scope string, tile uint32, country string, accepted bool) detect.Verdict {
	held := h.owner[tile]

	c := detect.Click{
		Scope:   scope,
		Tile:    tile,
		Country: country,
		At:      h.clock.Now(),
		Held:    held,
		NoOp:    held == country,
		Cleared: held != country && h.ground[tile] != "" && held == h.ground[tile],
	}

	verdict, _ := h.watchdog.Watch(c)

	if accepted {
		h.watchdog.Committed(c)
		switch {
		case c.Cleared:
			h.owner[tile] = ""
		case !c.NoOp:
			h.owner[tile] = country
		}
	}

	return verdict
}

func ms(values ...int) []time.Duration {
	out := make([]time.Duration, 0, len(values))
	for _, value := range values {
		out = append(out, time.Duration(value)*time.Millisecond)
	}
	return out
}

func strictConfig() retaker.Config {
	return retaker.Config{
		ReactionWindow: 2 * time.Second,
		MinReactions:   4,
		MaxMedian:      250 * time.Millisecond,
		MaxSpread:      120 * time.Millisecond,
		TrackWindow:    5 * time.Minute,
	}
}

func (h *harness) war(suspect string, first uint32, delays []time.Duration) detect.Verdict {
	var verdict detect.Verdict

	for i, delay := range delays {
		tile := first + uint32(i)

		h.click("player", tile, "FR")
		h.clock.Advance(delay)
		verdict = h.click(suspect, tile, "PS")

		h.clock.Advance(time.Second)
	}

	return verdict
}

func TestATightBandOfFastReactionsIsCertain(t *testing.T) {
	h := newHarness(strictConfig())

	verdict := h.war("bot", 100, ms(80, 85, 78, 90, 82, 88))

	assert.Equal(t, detect.Certain, verdict, "faster than a hand decides, and in a band no hand holds")
}

func TestATightBandAtAHumanTempoIsOnlySuspect(t *testing.T) {
	h := newHarness(strictConfig())

	verdict := h.war("bot", 200, ms(980, 1010, 995, 1020, 1000, 990))

	assert.Equal(t, detect.Suspect, verdict)
}

func TestAHumanTileWarIsClear(t *testing.T) {
	h := newHarness(strictConfig())

	verdict := h.war("defender", 300, ms(420, 900, 310, 1500, 640, 1100, 380, 780))

	assert.Equal(t, detect.Clear, verdict, "human reaction spread flags nobody")
}

func TestTheReactionIsTimedAndReported(t *testing.T) {
	h := newHarness(strictConfig())

	h.war("bot", 400, ms(80, 85, 78, 90))

	require.Len(t, h.reactions, 4)
	assert.Equal(t, 80*time.Millisecond, h.reactions[0])
}

func TestSpammingATileYouAlreadyOwnDoesNotFrameTheNextClicker(t *testing.T) {
	h := newHarness(strictConfig())

	const tile = uint32(500)

	h.click("bot", tile, "PS")

	for range 20 {
		h.clock.Advance(100 * time.Millisecond)
		h.click("bot", tile, "PS")
	}

	h.clock.Advance(50 * time.Millisecond)
	assert.Equal(t, detect.Clear, h.click("player", tile, "FR"))
	assert.Empty(t, h.reactions, "a no-op click is not part of an exchange")
}

func TestReactingToYourselfIsNotAReaction(t *testing.T) {
	h := newHarness(strictConfig())

	for i := range uint32(10) {
		tile := 600 + i

		h.click("bot", tile, "PS")
		h.clock.Advance(80 * time.Millisecond)
		h.click("bot", tile, "IL")

		h.clock.Advance(time.Second)
	}

	assert.Empty(t, h.reactions)
}

func TestASlowReactionIsOutsideTheWindow(t *testing.T) {
	config := strictConfig()
	config.ReactionWindow = 500 * time.Millisecond

	h := newHarness(config)

	h.war("other", 700, ms(3000, 3000, 3000, 3000))

	assert.Empty(t, h.reactions, "a re-take seconds later is not a reaction")
}

func TestARefusedClickCannotFrameAnHonestPlayer(t *testing.T) {
	h := newHarness(strictConfig())

	for i := range uint32(12) {
		tile := 800 + i

		require.Equal(t, detect.Clear, h.refused("griefer", tile, "zz"))

		h.clock.Advance(60 * time.Millisecond)
		h.click("player", tile, "FR")

		h.clock.Advance(time.Second)
	}

	assert.Empty(t, h.reactions, "a refused click takes no tile")
}

func TestADroppedClickCannotFrameAnHonestPlayer(t *testing.T) {
	h := newHarness(strictConfig())

	const contested = uint32(900)

	h.click("player", contested, "FR")

	h.clock.Advance(80 * time.Millisecond)
	h.refused("bot", contested, "PS")

	before := len(h.reactions)

	h.clock.Advance(80 * time.Millisecond)
	h.click("player", contested, "FR")

	assert.Len(t, h.reactions, before, "the player is not reacting to a click that never landed")
}

func TestReactionsAgeOutOfTheWindow(t *testing.T) {
	config := strictConfig()
	config.TrackWindow = time.Minute

	h := newHarness(config)

	require.Equal(t, detect.Certain, h.war("bot", 1000, ms(80, 85, 78, 90)))

	h.clock.Advance(2 * time.Minute)

	assert.Equal(t, detect.Clear, h.click("bot", 1100, "PS"),
		"stale reactions must not keep a verdict alive")
}

func productionConfig() retaker.Config {
	return retaker.Config{
		ReactionWindow: 5 * time.Second,
		MinReactions:   20,
		MaxSpread:      300 * time.Millisecond,
		MaxMedian:      250 * time.Millisecond,
		MinTiles:       15,
		RoamMedian:     600 * time.Millisecond,
		CertainTiles:   30,
		TrackWindow:    15 * time.Minute,
	}
}

func recaptureBot(n int) []time.Duration {
	cycle := ms(150, 200, 250, 280, 300, 350, 400, 600, 900, 120)

	delays := make([]time.Duration, 0, n)
	for i := range n {
		delays = append(delays, cycle[i%len(cycle)])
	}
	return delays
}

func TestTheRecaptureBotOfSeptember14IsCaught(t *testing.T) {
	h := newHarness(productionConfig())

	assert.Equal(t, detect.Suspect, h.war("bot", 2000, recaptureBot(20)),
		"too loose a band for the reflex rule, too fast across twenty tiles for a hand")

	assert.Equal(t, detect.Certain, h.war("bot", 3000, recaptureBot(10)))
}

func TestTheSameSpeedOnAFewTilesIsATileWar(t *testing.T) {
	h := newHarness(productionConfig())

	var verdict detect.Verdict
	for i, delay := range recaptureBot(60) {
		tile := 4000 + uint32(i%3)

		h.click("bot", tile, "BG")
		h.clock.Advance(delay)
		verdict = h.click("player", tile, "FR")

		h.clock.Advance(time.Second)
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestHumanReactionsAcrossManyTilesAreClear(t *testing.T) {
	h := newHarness(productionConfig())

	delays := make([]time.Duration, 0, 60)
	for range 6 {
		delays = append(delays, ms(700, 900, 1000, 1100, 1200, 1300, 1400, 1500, 1600, 800)...)
	}

	assert.Equal(t, detect.Clear, h.war("player", 5000, delays))
}

func TestRoamIsOffWithoutMinTiles(t *testing.T) {
	config := productionConfig()
	config.MinTiles = 0

	h := newHarness(config)

	assert.Equal(t, detect.Clear, h.war("bot", 6000, recaptureBot(40)))
}

func TestAClearIsSomethingToReactTo(t *testing.T) {
	h := newHarness(retaker.Config{})
	h.ground[7], h.owner[7] = "BG", "BG"

	h.click("attacker", 7, "FR")
	require.Empty(t, h.owner[7], "cleared, not taken")

	h.clock.Advance(800 * time.Millisecond)
	h.click("attacker", 7, "FR")
	assert.Empty(t, h.reactions, "taking the tile you emptied yourself is no reaction")

	h.clock.Advance(1200 * time.Millisecond)
	h.click("native", 7, "BG")
	assert.Equal(t, ms(1200), h.reactions, "BG winning its ground back reacts to the take")

	h.clock.Advance(900 * time.Millisecond)
	h.click("attacker", 7, "FR")
	h.clock.Advance(700 * time.Millisecond)
	h.click("native", 7, "BG")
	assert.Equal(t, ms(1200, 900, 700), h.reactions, "a clear is reacted to, and is a reaction itself")
}
