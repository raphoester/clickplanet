package jury_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// stubWatchdog says whatever the test tells it to, and remembers every click it
// was shown.
type stubWatchdog struct {
	name    string
	verdict detect.Verdict

	seen      int
	committed int
}

func (w *stubWatchdog) Name() string { return w.name }

func (w *stubWatchdog) Watch(detect.Click) (detect.Verdict, detect.Evidence) {
	w.seen++
	return w.verdict, detect.Evidence{
		Rule:   w.name + "-rule",
		Fields: []detect.Field{{Key: "seen", Value: w.seen}},
	}
}

func (w *stubWatchdog) Committed(detect.Click) { w.committed++ }

type harness struct {
	jury    *jury.Jury
	clock   *fakeClock
	reports []detect.Report
}

func newHarness(config jury.Config, ban shadowban.Config, watchdogs ...detect.Watchdog) *harness {
	h := &harness{clock: &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}}

	banner := shadowban.New(ban, h.clock)
	h.jury = jury.New(config, banner, h.clock, func(report detect.Report) {
		h.reports = append(h.reports, report)
	}, watchdogs...)

	return h
}

func (h *harness) click() bool {
	drop := h.jury.Inspect(detect.Click{
		Scope:   "caller",
		Tile:    1,
		Country: "FR",
		At:      h.clock.now,
	})

	h.clock.now = h.clock.now.Add(time.Second)

	return drop
}

func juryConfig() jury.Config {
	return jury.Config{
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
		&stubWatchdog{name: "sure", verdict: detect.Certain},
		&stubWatchdog{name: "quiet", verdict: detect.Clear},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestOneSuspectIsNotEnough(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "unsure", verdict: detect.Suspect},
		&stubWatchdog{name: "quiet", verdict: detect.Clear},
	)

	assert.False(t, h.click(), "a bound loose enough to be Suspect bans real players on its own")
	assert.Empty(t, h.reports)
}

func TestTwoSuspectsCrossIntoABan(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestMinSuspectsIsWhereTheLineIs(t *testing.T) {
	config := juryConfig()
	config.MinSuspects = 3

	h := newHarness(config, banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	assert.False(t, h.click())
}

func TestASuspicionOlderThanTheWindowStopsCounting(t *testing.T) {
	config := juryConfig()
	config.SuspicionWindow = time.Minute

	stale := &stubWatchdog{name: "stale", verdict: detect.Suspect}
	late := &stubWatchdog{name: "late", verdict: detect.Clear}

	h := newHarness(config, banConfig(), stale, late)

	require.False(t, h.click())

	stale.verdict = detect.Clear
	h.clock.now = h.clock.now.Add(5 * time.Minute)

	// The second watchdog only speaks up now, long after the first went quiet.
	// Two readings five minutes apart are not a caller doing two things at once.
	late.verdict = detect.Suspect
	assert.False(t, h.click())
}

func TestEveryWatchdogSeesEveryClickIncludingTheDroppedOnes(t *testing.T) {
	certain := &stubWatchdog{name: "sure", verdict: detect.Certain}
	other := &stubWatchdog{name: "other", verdict: detect.Clear}

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
		&stubWatchdog{name: "sure", verdict: detect.Certain},
		&stubWatchdog{name: "quiet", verdict: detect.Clear},
	)

	require.True(t, h.click())
	require.Len(t, h.reports, 1)

	report := h.reports[0]
	assert.Equal(t, "caller", report.Scope)
	assert.Equal(t, 1, report.Flags)
	assert.Equal(t, "FR", report.TopCountry)

	require.Len(t, report.Opinions, 2, "what did not fire is half of reading a ban that did")
	assert.Equal(t, detect.Certain, report.Opinions[0].Verdict)
	assert.Equal(t, detect.Clear, report.Opinions[1].Verdict)
}

func TestTheFlagIsNotRepeatedOnEveryClickInsideIt(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "sure", verdict: detect.Certain},
	)

	for range 60 {
		h.click()
	}

	assert.Len(t, h.reports, 1, "one report per reflag interval, not one per click")
}

func TestACallerThatKeepsAtItIsReportedAgain(t *testing.T) {
	ban := banConfig()
	ban.ReflagInterval = time.Minute

	h := newHarness(juryConfig(), ban, &stubWatchdog{name: "sure", verdict: detect.Certain})

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

	h := newHarness(juryConfig(), ban, &stubWatchdog{name: "sure", verdict: detect.Certain})

	assert.False(t, h.click(), "the mode to deploy in")
	assert.Len(t, h.reports, 1, "enforce must not change what is judged")
	assert.Equal(t, 1, h.jury.Flagged())
}

func TestCommittedReachesEveryWatchdog(t *testing.T) {
	first := &stubWatchdog{name: "first"}
	second := &stubWatchdog{name: "second"}

	h := newHarness(juryConfig(), banConfig(), first, second)

	h.jury.Committed(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.now})

	assert.Equal(t, 1, first.committed)
	assert.Equal(t, 1, second.committed)
}

func TestACallerWithNoScopeIsNotJudged(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Certain}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	assert.False(t, h.jury.Inspect(detect.Click{Tile: 1, Country: "FR", At: h.clock.now}))
	assert.Equal(t, 0, watchdog.seen)
}

func TestTheCallerFactsTravelWithTheBan(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Clear}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	for range 20 {
		h.click()
	}

	h.clock.now = h.clock.now.Add(20 * time.Minute)
	watchdog.verdict = detect.Certain

	require.True(t, h.click())
	require.Len(t, h.reports, 1)

	report := h.reports[0]
	assert.Equal(t, 21, report.Clicks)
	assert.Greater(t, report.ActiveFor, 20*time.Minute)
	assert.Greater(t, report.LongestGap, 19*time.Minute, "the break is visible in the line")
	assert.Equal(t, 21, report.TopCountryClicks)
}
