package retaker_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

type harness struct {
	watchdog  *retaker.Watchdog
	clock     *fakeClock
	owner     map[uint32]string
	reactions []time.Duration
}

func newHarness(config retaker.Config) *harness {
	h := &harness{
		clock: &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)},
		owner: map[uint32]string{},
	}

	h.watchdog = retaker.New(config, h.clock, func(d time.Duration) {
		h.reactions = append(h.reactions, d)
	})

	return h
}

// click walks one click through the watchdog the way the interceptor does: the
// tile's owner is read before the handler runs, and Committed only follows a
// click the handler accepted.
func (h *harness) click(scope string, tile uint32, country string) detect.Verdict {
	return h.deliver(scope, tile, country, true)
}

// refused is a click the handler turned down. It changed no tile, so nothing
// reports back.
func (h *harness) refused(scope string, tile uint32, country string) detect.Verdict {
	return h.deliver(scope, tile, country, false)
}

func (h *harness) deliver(scope string, tile uint32, country string, accepted bool) detect.Verdict {
	held := h.owner[tile]

	c := detect.Click{
		Scope:   scope,
		Tile:    tile,
		Country: country,
		At:      h.clock.now,
		Held:    held,
		NoOp:    held == country,
	}

	verdict, _ := h.watchdog.Watch(c)

	if accepted {
		h.watchdog.Committed(c)
		if !c.NoOp {
			h.owner[tile] = country
		}
	}

	return verdict
}

// ms spells the delays out in the unit they are actually argued about.
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

// war runs one exchange per delay: somebody takes the tile, the suspect takes it
// straight back.
func (h *harness) war(suspect string, first uint32, delays []time.Duration) detect.Verdict {
	var verdict detect.Verdict

	for i, delay := range delays {
		tile := first + uint32(i)

		h.click("player", tile, "FR")
		h.clock.advance(delay)
		verdict = h.click(suspect, tile, "PS")

		h.clock.advance(time.Second)
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

	// The bot actually seen in production: it picks a human-looking delay on
	// purpose, so it sails past MaxMedian. Regularity is all that is left, and
	// regularity alone must not ban on its own.
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
		h.clock.advance(100 * time.Millisecond)
		h.click("bot", tile, "PS")
	}

	h.clock.advance(50 * time.Millisecond)
	assert.Equal(t, detect.Clear, h.click("player", tile, "FR"))
	assert.Empty(t, h.reactions, "a no-op click is not part of an exchange")
}

func TestReactingToYourselfIsNotAReaction(t *testing.T) {
	h := newHarness(strictConfig())

	for i := range uint32(10) {
		tile := 600 + i

		h.click("bot", tile, "PS")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "IL")

		h.clock.advance(time.Second)
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

		h.clock.advance(60 * time.Millisecond)
		h.click("player", tile, "FR")

		h.clock.advance(time.Second)
	}

	assert.Empty(t, h.reactions, "a refused click takes no tile")
}

func TestADroppedClickCannotFrameAnHonestPlayer(t *testing.T) {
	h := newHarness(strictConfig())

	const contested = uint32(900)

	h.click("player", contested, "FR")

	// A banned caller's click is answered OK and never reaches the map, so the
	// jury never calls Committed for it. The bot reacting to the player is real
	// and is counted; what must not happen is the reverse.
	h.clock.advance(80 * time.Millisecond)
	h.refused("bot", contested, "PS")

	before := len(h.reactions)

	h.clock.advance(80 * time.Millisecond)
	h.click("player", contested, "FR")

	assert.Len(t, h.reactions, before, "the player is not reacting to a click that never landed")
}

func TestReactionsAgeOutOfTheWindow(t *testing.T) {
	config := strictConfig()
	config.TrackWindow = time.Minute

	h := newHarness(config)

	require.Equal(t, detect.Certain, h.war("bot", 1000, ms(80, 85, 78, 90)))

	h.clock.advance(2 * time.Minute)

	assert.Equal(t, detect.Clear, h.click("bot", 1100, "PS"),
		"stale reactions must not keep a verdict alive")
}
