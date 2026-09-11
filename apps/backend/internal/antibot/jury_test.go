package antibot_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// stubWatchdog says whatever the test tells it to, and remembers every click it
// was shown.
type stubWatchdog struct {
	name    string
	verdict antibot.Verdict

	seen      int
	committed int
}

func (w *stubWatchdog) Name() string { return w.name }

func (w *stubWatchdog) Watch(antibot.Click) (antibot.Verdict, antibot.Evidence) {
	w.seen++
	return w.verdict, antibot.Evidence{
		Rule:   w.name + "-rule",
		Fields: []antibot.Field{{Key: "seen", Value: w.seen}},
	}
}

func (w *stubWatchdog) Committed(antibot.Click) { w.committed++ }

type harness struct {
	jury    *antibot.Jury
	clock   *fakeClock
	reports []antibot.Report
}

func newHarness(config antibot.Config, ban shadowban.Config, watchdogs ...antibot.Watchdog) *harness {
	h := &harness{clock: &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}}

	banner := shadowban.New(ban, h.clock)
	h.jury = antibot.NewJury(config, banner, h.clock, func(report antibot.Report) {
		h.reports = append(h.reports, report)
	}, watchdogs...)

	return h
}

func (h *harness) click() bool {
	drop := h.jury.Inspect(antibot.Click{
		Scope:   "caller",
		Tile:    1,
		Country: "FR",
		At:      h.clock.now,
	})

	h.clock.now = h.clock.now.Add(time.Second)

	return drop
}

func juryConfig() antibot.Config {
	return antibot.Config{
		MinSuspects:     2,
		SuspicionWindow: 10 * time.Minute,
		TrackWindow:     time.Hour,
	}
}

func banConfig() shadowban.Config {
	return shadowban.Config{
		Enforce:        true,
		BanDuration:    time.Hour,
		ReflagInterval: 5 * time.Minute,
	}
}

func TestOneCertainWatchdogBansOnItsOwn(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "sure", verdict: antibot.Certain},
		&stubWatchdog{name: "quiet", verdict: antibot.Clear},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestOneSuspectIsNotEnough(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "unsure", verdict: antibot.Suspect},
		&stubWatchdog{name: "quiet", verdict: antibot.Clear},
	)

	assert.False(t, h.click(), "a bound loose enough to be Suspect bans real players on its own")
	assert.Empty(t, h.reports)
}

func TestTwoSuspectsCrossIntoABan(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: antibot.Suspect},
		&stubWatchdog{name: "second", verdict: antibot.Suspect},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestMinSuspectsIsWhereTheLineIs(t *testing.T) {
	config := juryConfig()
	config.MinSuspects = 3

	h := newHarness(config, banConfig(),
		&stubWatchdog{name: "first", verdict: antibot.Suspect},
		&stubWatchdog{name: "second", verdict: antibot.Suspect},
	)

	assert.False(t, h.click())
}

func TestASuspicionOlderThanTheWindowStopsCounting(t *testing.T) {
	config := juryConfig()
	config.SuspicionWindow = time.Minute

	stale := &stubWatchdog{name: "stale", verdict: antibot.Suspect}
	late := &stubWatchdog{name: "late", verdict: antibot.Clear}

	h := newHarness(config, banConfig(), stale, late)

	require.False(t, h.click())

	stale.verdict = antibot.Clear
	h.clock.now = h.clock.now.Add(5 * time.Minute)

	// The second watchdog only speaks up now, long after the first went quiet.
	// Two readings five minutes apart are not a caller doing two things at once.
	late.verdict = antibot.Suspect
	assert.False(t, h.click())
}

func TestEveryWatchdogSeesEveryClickIncludingTheDroppedOnes(t *testing.T) {
	certain := &stubWatchdog{name: "sure", verdict: antibot.Certain}
	other := &stubWatchdog{name: "other", verdict: antibot.Clear}

	h := newHarness(juryConfig(), banConfig(), certain, other)

	for range 10 {
		h.click()
	}

	// A watchdog cut off the moment another one banned the caller would be
	// judging a caller that appears to have stopped clicking.
	assert.Equal(t, 10, certain.seen)
	assert.Equal(t, 10, other.seen)
}

func TestTheReportNamesEveryWatchdogIncludingTheQuietOnes(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "sure", verdict: antibot.Certain},
		&stubWatchdog{name: "quiet", verdict: antibot.Clear},
	)

	require.True(t, h.click())
	require.Len(t, h.reports, 1)

	report := h.reports[0]
	assert.Equal(t, "caller", report.Scope)
	assert.Equal(t, 1, report.Flags)
	assert.Equal(t, "FR", report.TopCountry)

	require.Len(t, report.Opinions, 2, "what did not fire is half of reading a ban that did")
	assert.Equal(t, antibot.Certain, report.Opinions[0].Verdict)
	assert.Equal(t, antibot.Clear, report.Opinions[1].Verdict)
}

func TestTheFlagIsNotRepeatedOnEveryClickInsideIt(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "sure", verdict: antibot.Certain},
	)

	for range 60 {
		h.click()
	}

	assert.Len(t, h.reports, 1, "one report per reflag interval, not one per click")
}

func TestACallerThatKeepsAtItIsReportedAgain(t *testing.T) {
	ban := banConfig()
	ban.ReflagInterval = time.Minute

	h := newHarness(juryConfig(), ban, &stubWatchdog{name: "sure", verdict: antibot.Certain})

	for range 4 {
		h.click()
		h.clock.now = h.clock.now.Add(time.Minute)
	}

	require.Len(t, h.reports, 4)
	for i, report := range h.reports {
		assert.Equal(t, i+1, report.Flags, "a rising count is independent evidence repeating")
	}
}

func TestEnforceOffJudgesAndDropsNothing(t *testing.T) {
	ban := banConfig()
	ban.Enforce = false

	h := newHarness(juryConfig(), ban, &stubWatchdog{name: "sure", verdict: antibot.Certain})

	assert.False(t, h.click(), "the mode to deploy in")
	assert.Len(t, h.reports, 1, "enforce must not change what is judged")
	assert.Equal(t, 1, h.jury.Flagged())
}

func TestCommittedReachesEveryWatchdog(t *testing.T) {
	first := &stubWatchdog{name: "first"}
	second := &stubWatchdog{name: "second"}

	h := newHarness(juryConfig(), banConfig(), first, second)

	h.jury.Committed(antibot.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.now})

	assert.Equal(t, 1, first.committed)
	assert.Equal(t, 1, second.committed)
}

func TestACallerWithNoScopeIsNotJudged(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: antibot.Certain}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	assert.False(t, h.jury.Inspect(antibot.Click{Tile: 1, Country: "FR", At: h.clock.now}))
	assert.Equal(t, 0, watchdog.seen)
}

func TestTheCallerFactsTravelWithTheBan(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: antibot.Clear}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	for range 20 {
		h.click()
	}

	h.clock.now = h.clock.now.Add(20 * time.Minute)
	watchdog.verdict = antibot.Certain

	require.True(t, h.click())
	require.Len(t, h.reports, 1)

	report := h.reports[0]
	assert.Equal(t, 21, report.Clicks)
	assert.Greater(t, report.ActiveFor, 20*time.Minute)
	assert.Greater(t, report.LongestGap, 19*time.Minute, "the break is visible in the line")
	assert.Equal(t, 21, report.TopCountryClicks)
}
