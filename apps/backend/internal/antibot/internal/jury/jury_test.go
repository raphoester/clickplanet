package jury_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// stubWatchdog says whatever the test tells it to, and remembers every click it
// was shown.
type stubWatchdog struct {
	name    string
	verdict detect.Verdict

	seen      int
	committed int
	attempted int
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

func (w *stubWatchdog) Attempted(detect.Click) { w.attempted++ }

type harness struct {
	jury    *jury.Jury
	clock   *cptime.FixedClock
	reports []detect.Report
	rises   []string
}

func newHarness(config jury.Config, ban shadowban.Config, watchdogs ...detect.Watchdog) *harness {
	h := &harness{clock: cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))}

	banner := shadowban.NewBans(ban, h.clock, shadowban.NewMemoryPersistence(), shadowban.NewMemoryPersistence(), func(error) {})
	h.jury = jury.New(config, banner, h.clock, jury.Hooks{
		OnFlag: func(report detect.Report) {
			h.reports = append(h.reports, report)
		},
		OnRise: func(watchdog string, level detect.Verdict) {
			h.rises = append(h.rises, watchdog+" "+level.String())
		},
	}, watchdogs...)

	return h
}

func (h *harness) click() bool {
	drop := h.jury.Inspect(detect.Click{
		Scope:   "caller",
		Tile:    1,
		Country: "FR",
		At:      h.clock.Now(),
	})

	h.clock.Advance(time.Second)

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
		BanDurations:   []time.Duration{time.Hour},
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
	h.clock.Advance(5 * time.Minute)

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
		h.clock.Advance(time.Minute)
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

	h.jury.Committed(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Equal(t, 1, first.committed)
	assert.Equal(t, 1, second.committed)
}

func TestAttemptedReachesEveryWatchdogAndJudgesNothing(t *testing.T) {
	first := &stubWatchdog{name: "first", verdict: detect.Certain}
	second := &stubWatchdog{name: "second"}

	h := newHarness(juryConfig(), banConfig(), first, second)

	h.jury.Attempted(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})
	h.jury.Attempted(detect.Click{Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Equal(t, 1, first.attempted, "a click with no scope is nobody's")
	assert.Equal(t, 1, second.attempted)
	assert.Equal(t, 0, first.seen, "a try the throttle may still refuse is never a verdict")
	assert.Empty(t, h.reports)
}

func TestACallerWithNoScopeIsNotJudged(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Certain}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	assert.False(t, h.jury.Inspect(detect.Click{Tile: 1, Country: "FR", At: h.clock.Now()}))
	assert.Equal(t, 0, watchdog.seen)
}

func TestTheCallerFactsTravelWithTheBan(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Clear}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	for range 20 {
		h.click()
	}

	h.clock.Advance(20 * time.Minute)
	watchdog.verdict = detect.Certain

	require.True(t, h.click())
	require.Len(t, h.reports, 1)

	report := h.reports[0]
	assert.Equal(t, 21, report.Clicks)
	assert.Greater(t, report.ActiveFor, 20*time.Minute)
	assert.Greater(t, report.LongestGap, 19*time.Minute, "the break is visible in the line")
	assert.Equal(t, 21, report.TopCountryClicks)
}

func TestExaminingAnUnknownCallerListsEveryWatchdogAsClear(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Certain},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	examination := h.jury.Examine("stranger")

	assert.False(t, examination.Tracked)
	assert.False(t, examination.Guilty)
	assert.Equal(t, 2, examination.MinSuspects)
	assert.Equal(t, []detect.Reading{
		{Watchdog: "first", Level: "clear"},
		{Watchdog: "second", Level: "clear"},
	}, examination.Readings)
}

func TestExaminingSaysHowCloseACallerIsAndChangesNothing(t *testing.T) {
	unsure := &stubWatchdog{name: "unsure", verdict: detect.Suspect}
	quiet := &stubWatchdog{name: "quiet", verdict: detect.Clear}

	h := newHarness(juryConfig(), banConfig(), unsure, quiet)

	first := h.clock.Now()
	require.False(t, h.click())
	require.False(t, h.click())
	last := first.Add(time.Second)

	examination := h.jury.Examine("caller")

	assert.Equal(t, detect.Examination{
		Scope:   "caller",
		Tracked: true,
		Readings: []detect.Reading{
			{Watchdog: "unsure", Level: "suspect", Evidence: "unsure-rule seen=2", At: last},
			{Watchdog: "quiet", Level: "clear", Evidence: "quiet-rule seen=2", At: last},
		},
		Suspects:         1,
		MinSuspects:      2,
		Clicks:           2,
		ActiveFor:        time.Second,
		LongestGap:       time.Second,
		LastClickAt:      last,
		TopCountry:       "FR",
		TopCountryClicks: 2,
	}, examination)

	assert.Equal(t, 2, unsure.seen, "examining asks no watchdog again")
	assert.Empty(t, h.reports)
	assert.Zero(t, h.jury.Flagged())
}

func TestExaminingWithTwoSuspectsReadsGuiltyWithoutBanning(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	// The first click bans; the examination afterwards must not ban a second time.
	require.True(t, h.click())

	examination := h.jury.Examine("caller")

	assert.True(t, examination.Guilty)
	assert.Equal(t, 2, examination.Suspects)
	assert.Len(t, h.reports, 1)
}

func TestExaminingAgesASuspicionPastTheWindow(t *testing.T) {
	h := newHarness(juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	require.True(t, h.click())
	h.clock.Advance(11 * time.Minute)

	examination := h.jury.Examine("caller")

	assert.False(t, examination.Guilty)
	assert.Zero(t, examination.Suspects)
	assert.Equal(t, "clear", examination.Readings[0].Level)
	assert.Equal(t, "first-rule seen=1", examination.Readings[0].Evidence, "the evidence stays to read")
}

func TestAReadingFlappingAcrossABoundRisesOncePerWindow(t *testing.T) {
	watchdog := &stubWatchdog{name: "unsure", verdict: detect.Suspect}

	h := newHarness(juryConfig(), banConfig(), watchdog)

	// Suspect, clear, suspect … for five minutes: one standing suspicion, not one per click.
	for i := range 300 {
		watchdog.verdict = detect.Verdict(i % 2)
		h.click()
	}

	assert.Equal(t, []string{"unsure suspect"}, h.rises, "a counter per click would say how often it was asked, not how close it came")

	watchdog.verdict = detect.Clear
	h.click()
	h.clock.Advance(11 * time.Minute)

	watchdog.verdict = detect.Suspect
	h.click()

	assert.Equal(t, []string{"unsure suspect", "unsure suspect"}, h.rises, "past the window the suspicion had lapsed, so this is a new one")
}

func TestGoingStraightToCertainRisesThroughSuspect(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Clear}
	quiet := &stubWatchdog{name: "quiet", verdict: detect.Clear}

	h := newHarness(juryConfig(), banConfig(), watchdog, quiet)

	h.click()
	assert.Empty(t, h.rises, "clear is not a level")

	watchdog.verdict = detect.Certain
	h.click()
	h.click()

	assert.Equal(t, []string{"sure suspect", "sure certain"}, h.rises, "levels are cumulative, like histogram buckets")

	watchdog.verdict = detect.Suspect
	h.click()

	assert.Len(t, h.rises, 2, "falling back to suspect is not a rise")
}

func TestAGuestBannedByTheJuryCannotShedItWithAFreshCookie(t *testing.T) {
	sure := &stubWatchdog{name: "sure", verdict: detect.Certain}
	h := newHarness(juryConfig(), banConfig(), sure)

	require.True(t, h.jury.Inspect(detect.Click{Scope: "caller", Account: "guest", Tile: 1, Country: "FR", At: h.clock.Now()}))
	require.Len(t, h.reports, 1)
	assert.Equal(t, "guest", h.reports[0].Account)

	sure.verdict = detect.Clear
	h.clock.Advance(time.Second)

	assert.True(t, h.jury.Inspect(detect.Click{Scope: "caller", Account: "fresh-cookie", Tile: 1, Country: "FR", At: h.clock.Now()}),
		"the ban fell on the guest's scope too")
	assert.True(t, h.jury.Inspect(detect.Click{Scope: "elsewhere", Account: "guest", Tile: 1, Country: "FR", At: h.clock.Now()}),
		"and on the account, wherever it clicks from")
}

func TestASignedInAccountBannedByTheJuryLeavesItsScopeAlone(t *testing.T) {
	sure := &stubWatchdog{name: "sure", verdict: detect.Certain}
	h := newHarness(juryConfig(), banConfig(), sure)

	require.True(t, h.jury.Inspect(detect.Click{Scope: "campus", Account: "bot", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	sure.verdict = detect.Clear
	h.clock.Advance(time.Second)

	assert.False(t, h.jury.Inspect(detect.Click{Scope: "campus", Account: "student", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))
}
