package shadowban_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/shadowban"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

type fakeOwner map[uint32]string

func (o fakeOwner) Owner(tile uint32) (string, bool) {
	value, ok := o[tile]
	return value, ok
}

type harness struct {
	detector  *shadowban.Detector
	clock     *fakeClock
	owner     fakeOwner
	reactions []time.Duration
	flags     map[string]shadowban.Report
}

func newHarness(t *testing.T, config shadowban.Config) *harness {
	t.Helper()

	h := &harness{
		clock: &fakeClock{now: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)},
		owner: fakeOwner{},
		flags: map[string]shadowban.Report{},
	}

	h.detector = shadowban.New(
		config,
		h.owner,
		h.clock,
		func(d time.Duration) { h.reactions = append(h.reactions, d) },
		func(scope string, report shadowban.Report) { h.flags[scope] = report },
	)

	return h
}

// click paints tile for country and keeps the fake map in step, so that the
// no-op rule sees the same world the detector does.
func (h *harness) click(scope string, tile uint32, country string) bool {
	drop, takes := h.detector.Observe(scope, tile, country)
	if !drop && takes {
		h.detector.Took(scope, tile)
		h.owner[tile] = country
	}
	return drop
}

func strictConfig() shadowban.Config {
	return shadowban.Config{
		Enforce:        true,
		ReactionWindow: 2 * time.Second,
		MinReactions:   4,
		MaxMedian:      250 * time.Millisecond,
		MaxSpread:      120 * time.Millisecond,
		TrackWindow:    5 * time.Minute,
		BanDuration:    time.Hour,
		ReflagInterval: 5 * time.Minute,
	}
}

func TestABotThatAnswersInATightBandIsFlagged(t *testing.T) {
	h := newHarness(t, strictConfig())

	delays := []time.Duration{80, 85, 78, 90, 82, 88}

	for i, delay := range delays {
		tile := uint32(100 + i)

		h.click("player", tile, "FR")
		h.clock.advance(delay * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	report, flagged := h.flags["bot"]
	require.True(t, flagged, "a tight band of fast reactions should flag")

	// The report is the evidence at the moment the threshold was crossed, so it
	// holds MinReactions of them and not the whole run.
	assert.Equal(t, 4, report.Reactions)
	assert.Equal(t, "PS", report.TopCountry)
	assert.Equal(t, 4, report.TopCountryClicks)
	assert.LessOrEqual(t, report.Median, 250*time.Millisecond)
	assert.LessOrEqual(t, report.Spread, 120*time.Millisecond)

	assert.NotContains(t, h.flags, "player", "the player it reacted to is not the bot")
}

func TestAHumanTileWarIsNotFlagged(t *testing.T) {
	h := newHarness(t, strictConfig())

	// Fast, but nothing like regular: a hand does not hold a 10 ms band.
	delays := []time.Duration{420, 900, 310, 1500, 640, 1100, 380, 780}

	for i, delay := range delays {
		tile := uint32(200 + i)

		h.click("attacker", tile, "IL")
		h.clock.advance(delay * time.Millisecond)
		h.click("defender", tile, "PS")

		h.clock.advance(time.Second)
	}

	assert.Empty(t, h.flags, "human reaction spread should not flag anyone")
}

func TestSpammingATileYouAlreadyOwnDoesNotFrameTheNextClicker(t *testing.T) {
	h := newHarness(t, strictConfig())

	const tile = uint32(300)

	h.click("bot", tile, "PS")

	// The bot keeps clicking a tile it already holds. Each one is a no-op, so
	// none of them may become the event somebody else looks like a reaction to.
	for range 20 {
		h.clock.advance(100 * time.Millisecond)
		h.click("bot", tile, "PS")
	}

	h.clock.advance(50 * time.Millisecond)
	h.click("player", tile, "FR")

	assert.Empty(t, h.reactions, "a no-op click is not part of an exchange")
	assert.Empty(t, h.flags)
}

func TestReactingToYourselfIsNotAReaction(t *testing.T) {
	h := newHarness(t, strictConfig())

	for i := range 10 {
		tile := uint32(400 + i)

		h.click("bot", tile, "PS")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "IL")

		h.clock.advance(time.Second)
	}

	assert.Empty(t, h.reactions)
	assert.Empty(t, h.flags)
}

func TestASlowReactionIsOutsideTheWindow(t *testing.T) {
	config := strictConfig()
	config.ReactionWindow = 500 * time.Millisecond

	h := newHarness(t, config)

	for i := range 10 {
		tile := uint32(500 + i)

		h.click("player", tile, "FR")
		h.clock.advance(3 * time.Second)
		h.click("other", tile, "PS")
	}

	assert.Empty(t, h.reactions, "a re-take minutes later is not a reaction")
}

func TestAFlaggedCallerIsDroppedSilentlyAndOnlyWhileEnforcing(t *testing.T) {
	for _, enforce := range []bool{true, false} {
		config := strictConfig()
		config.Enforce = enforce

		h := newHarness(t, config)

		var dropped bool
		for i := range 8 {
			tile := uint32(600 + i)

			h.click("player", tile, "FR")
			h.clock.advance(80 * time.Millisecond)
			if h.click("bot", tile, "PS") {
				dropped = true
			}

			h.clock.advance(time.Second)
		}

		assert.Contains(t, h.flags, "bot", "enforce must not change what is detected")
		assert.Equal(t, enforce, dropped, "only enforcing drops clicks")
	}
}

func TestTheFlagIsNotReportedOnEveryClickInsideIt(t *testing.T) {
	config := strictConfig()

	h := newHarness(t, config)

	var flags int
	h.detector = shadowban.New(config, h.owner, h.clock,
		nil,
		func(string, shadowban.Report) { flags++ },
	)

	for i := range 30 {
		tile := uint32(700 + i)

		h.click("player", tile, "FR")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	assert.Equal(t, 1, flags, "one flag per reflagInterval, not one per click")
}

func TestACallerThatKeepsAtItIsFlaggedAgain(t *testing.T) {
	config := strictConfig()
	config.ReflagInterval = time.Minute

	h := newHarness(t, config)

	var reports []shadowban.Report
	h.detector = shadowban.New(config, h.owner, h.clock, nil,
		func(_ string, report shadowban.Report) { reports = append(reports, report) },
	)

	tile := uint32(1700)
	for range 4 {
		for range 4 {
			tile++
			h.click("player", tile, "FR")
			h.clock.advance(80 * time.Millisecond)
			h.click("bot", tile, "PS")
			h.clock.advance(time.Second)
		}
		h.clock.advance(time.Minute)
	}

	require.Len(t, reports, 4, "each reflagInterval that still looks the same reports again")

	for i, report := range reports {
		assert.Equal(t, i+1, report.Flags, "the count rises so repeat evidence is legible")
	}

	assert.Greater(t, reports[3].ActiveFor, 3*time.Minute)
	assert.Less(t, reports[3].LongestGap, 2*time.Minute)
}

func TestPersistenceSeparatesASittingCallerFromOneThatLeaves(t *testing.T) {
	config := strictConfig()
	config.TrackWindow = time.Hour
	config.ReflagInterval = time.Minute

	h := newHarness(t, config)

	tile := uint32(1800)
	war := func() {
		for range 4 {
			tile++
			h.click("bot", tile, "PS")
			h.clock.advance(80 * time.Millisecond)
			h.click("player", tile, "FR")
			h.clock.advance(time.Second)
		}
	}

	war()

	// The player walks away for twenty minutes and comes back, as people do.
	h.clock.advance(20 * time.Minute)
	war()

	report := h.flags["player"]
	assert.Greater(t, report.LongestGap, 19*time.Minute, "the break is visible in the line")
	assert.Greater(t, report.ActiveFor, 20*time.Minute)
}

func TestTopCountryNamesWhatTheCallerPaintsMost(t *testing.T) {
	h := newHarness(t, strictConfig())

	for i := range 4 {
		h.click("bot", uint32(800+i), "IL")
	}
	for i := range 11 {
		h.click("bot", uint32(900+i), "PS")
	}

	for i := range 6 {
		tile := uint32(1000 + i)

		h.click("player", tile, "FR")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	report := h.flags["bot"]
	assert.Equal(t, "PS", report.TopCountry)
	assert.Equal(t, 15, report.TopCountryClicks, "11 before the war plus the 4 that flagged it")
	assert.Equal(t, 19, report.Clicks)
}

func TestADroppedClickDoesNotFrameTheNextClicker(t *testing.T) {
	h := newHarness(t, strictConfig())

	for i := range 8 {
		tile := uint32(1200 + i)

		h.click("player", tile, "FR")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	require.Equal(t, 1, h.detector.Flagged())

	const contested = uint32(1300)
	h.click("player", contested, "FR")

	h.clock.advance(80 * time.Millisecond)
	require.True(t, h.click("bot", contested, "PS"), "the bot is banned by now")

	before := len(h.reactions)

	// The bot's click was dropped, so the tile is still the player's and this is
	// a no-op rather than the player reacting to a take that never happened.
	h.clock.advance(80 * time.Millisecond)
	h.click("player", contested, "FR")

	assert.Equal(t, before, len(h.reactions))
	assert.NotContains(t, h.flags, "player")
}

func TestARefusedClickCannotFrameAnHonestPlayer(t *testing.T) {
	h := newHarness(t, strictConfig())

	// A griefer spams tiles with clicks the handler will refuse — an invalid
	// country, a tile out of range. Took is never called for those, so a player
	// clicking the same tile shortly after is reacting to nothing.
	for i := range 12 {
		tile := uint32(1400 + i)

		drop, _ := h.detector.Observe("griefer", tile, "zz")
		require.False(t, drop)

		h.clock.advance(60 * time.Millisecond)
		h.click("player", tile, "FR")

		h.clock.advance(time.Second)
	}

	assert.Empty(t, h.reactions, "a refused click takes no tile")
	assert.NotContains(t, h.flags, "player")
}

func TestALapsedBanIsNotHandedStraightBackOnStaleEvidence(t *testing.T) {
	config := strictConfig()
	config.TrackWindow = time.Minute
	config.BanDuration = 10 * time.Minute

	h := newHarness(t, config)

	for i := range 8 {
		tile := uint32(1500 + i)

		h.click("player", tile, "FR")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	require.Equal(t, 1, h.detector.Flagged())

	// The reactions that earned the ban are still on the caller when it lapses.
	h.clock.advance(11 * time.Minute)
	require.Equal(t, 0, h.detector.Flagged())

	assert.False(t, h.click("bot", 1600, "PS"), "the click lands")
	assert.Equal(t, 0, h.detector.Flagged(), "and does not earn a fresh ban on its own")
}

func TestABanOutlivesTheReactionsThatEarnedIt(t *testing.T) {
	config := strictConfig()
	config.TrackWindow = time.Minute
	config.BanDuration = time.Hour

	h := newHarness(t, config)

	for i := range 8 {
		tile := uint32(1100 + i)

		h.click("player", tile, "FR")
		h.clock.advance(80 * time.Millisecond)
		h.click("bot", tile, "PS")

		h.clock.advance(time.Second)
	}

	require.Contains(t, h.flags, "bot")
	require.Equal(t, 1, h.detector.Flagged())

	h.clock.advance(30 * time.Minute)
	assert.Equal(t, 1, h.detector.Flagged(), "the ban runs past the tracking window")
	assert.True(t, h.click("bot", 1150, "PS"), "and is still dropping clicks")

	h.clock.advance(31 * time.Minute)
	assert.Equal(t, 0, h.detector.Flagged())
	assert.False(t, h.click("bot", 1151, "PS"), "once it lapses, clicks land again")
}
