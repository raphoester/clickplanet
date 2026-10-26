package jury_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubWatchdog struct {
	name    string
	verdict detect.Verdict
	zeal    bool
	turns   bool

	accomplices []detect.Accomplice

	seen      int
	committed int
	attempted int
}

func (w *stubWatchdog) Name() string { return w.name }

func (w *stubWatchdog) Watch(detect.Click) (detect.Verdict, detect.Evidence) {
	w.seen++
	return w.verdict, detect.Evidence{
		Rule:        w.name + "-rule",
		Fields:      []detect.Field{{Key: "seen", Value: w.seen}},
		Zeal:        w.zeal,
		Turns:       w.turns,
		Accomplices: w.accomplices,
	}
}

func (w *stubWatchdog) Committed(detect.Click) { w.committed++ }

func (w *stubWatchdog) Attempted(detect.Click) { w.attempted++ }

type harness struct {
	t           *testing.T
	jury        *jury.Jury
	banner      *bans.Book
	convictions *convictions.MemoryStore
	clock       *cptime.FixedClock
	reports     []detect.Report
	rises       []string
}

func newHarness(t *testing.T, config jury.Config, ban bans.Config, watchdogs ...detect.Watchdog) *harness {
	t.Helper()
	h := &harness{
		t:           t,
		clock:       cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)),
		convictions: convictions.NewMemoryStore(),
	}

	h.banner = bans.NewBook(ban, pardon, h.clock, bans.NewMemoryStore(), bans.NewMemoryStore(), cpatomic.FakeTransactioner{}, &bans.MemoryEnforcer{})
	h.jury = jury.New(config, h.banner, h.convictions, h.clock, jury.Hooks{
		OnFlag: func(report detect.Report) {
			h.reports = append(h.reports, report)
		},
		OnRise: func(watchdog string, level detect.Verdict) {
			h.rises = append(h.rises, watchdog+" "+level.String())
		},
	}, watchdogs...)

	return h
}

func (h *harness) inspect(click detect.Click) bool {
	h.t.Helper()
	require.NoError(h.t, h.jury.Inspect(h.t.Context(), click))

	_, banned, err := h.banner.EnforcedSentence(h.t.Context(),
		bans.Caller{Scope: click.Scope, Account: click.Account, SignedIn: click.SignedIn})
	require.NoError(h.t, err)
	return banned
}

func (h *harness) flagged() int {
	h.t.Helper()
	flagged, err := h.jury.Flagged(h.t.Context())
	require.NoError(h.t, err)
	return flagged
}

func (h *harness) click() bool {
	drop := h.inspect(detect.Click{
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

const pardon = 72 * time.Hour

func banConfig() bans.Config {
	return bans.Config{
		Enforce:        true,
		BanDurations:   []time.Duration{time.Hour},
		ReflagInterval: 5 * time.Minute,
	}
}

func TestOneCertainWatchdogBansOnItsOwn(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "sure", verdict: detect.Certain},
		&stubWatchdog{name: "quiet", verdict: detect.Clear},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestOneSuspectIsNotEnough(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "unsure", verdict: detect.Suspect},
		&stubWatchdog{name: "quiet", verdict: detect.Clear},
	)

	assert.False(t, h.click(), "a bound loose enough to be Suspect bans real players on its own")
	assert.Empty(t, h.reports)
}

func TestTwoSuspectsCrossIntoABan(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestTwoReadingsOfZealAreOneSuspect(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "hours", verdict: detect.Suspect, zeal: true},
		&stubWatchdog{name: "retakes", verdict: detect.Suspect, zeal: true},
	)

	assert.False(t, h.click(), "a person who plays hard reads both")
	assert.Empty(t, h.reports)

	examination := h.jury.Examine("caller", "")
	assert.Equal(t, 1, examination.Suspects)
	assert.False(t, examination.Guilty)
}

func TestZealStillCrossesAnotherWatchdog(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "hours", verdict: detect.Suspect, zeal: true},
		&stubWatchdog{name: "retakes", verdict: detect.Suspect, zeal: true},
		&stubWatchdog{name: "cadence", verdict: detect.Suspect},
	)

	assert.True(t, h.click())
	require.Len(t, h.reports, 1)
}

func TestAReadingOfTurnsNeverCounts(t *testing.T) {
	turns := &stubWatchdog{name: "turns", verdict: detect.Suspect, turns: true}
	h := newHarness(t, juryConfig(), banConfig(), turns, &stubWatchdog{name: "cadence", verdict: detect.Suspect})

	assert.False(t, h.click(), "a person takes turns to beat the rate limit")

	turns.verdict = detect.Certain
	assert.False(t, h.click(), "at any level")
	assert.Empty(t, h.reports)

	examination := h.jury.Examine("caller", "")
	assert.Equal(t, "certain", examination.Readings[0].Level, "it still reads")
	assert.Equal(t, 1, examination.Suspects)
	assert.False(t, examination.Guilty)
}

func TestMinSuspectsIsWhereTheLineIs(t *testing.T) {
	config := juryConfig()
	config.MinSuspects = 3

	h := newHarness(t, config, banConfig(),
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

	h := newHarness(t, config, banConfig(), stale, late)

	require.False(t, h.click())

	stale.verdict = detect.Clear
	h.clock.Advance(5 * time.Minute)

	late.verdict = detect.Suspect
	assert.False(t, h.click())
}

func TestEveryWatchdogSeesEveryClickIncludingTheDroppedOnes(t *testing.T) {
	certain := &stubWatchdog{name: "sure", verdict: detect.Certain}
	other := &stubWatchdog{name: "other", verdict: detect.Clear}

	h := newHarness(t, juryConfig(), banConfig(), certain, other)

	for range 10 {
		h.click()
	}

	assert.Equal(t, 10, certain.seen)
	assert.Equal(t, 10, other.seen)
}

func TestTheReportNamesEveryWatchdogIncludingTheQuietOnes(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
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
	h := newHarness(t, juryConfig(), banConfig(),
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

	h := newHarness(t, juryConfig(), ban, &stubWatchdog{name: "sure", verdict: detect.Certain})

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

	h := newHarness(t, juryConfig(), ban, &stubWatchdog{name: "sure", verdict: detect.Certain})

	assert.False(t, h.click(), "the mode to deploy in")
	assert.Len(t, h.reports, 1, "enforce must not change what is judged")
	assert.Equal(t, 1, h.flagged())
}

func TestCommittedReachesEveryWatchdog(t *testing.T) {
	first := &stubWatchdog{name: "first"}
	second := &stubWatchdog{name: "second"}

	h := newHarness(t, juryConfig(), banConfig(), first, second)

	h.jury.Committed(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Equal(t, 1, first.committed)
	assert.Equal(t, 1, second.committed)
}

func TestAttemptedReachesEveryWatchdogAndJudgesNothing(t *testing.T) {
	first := &stubWatchdog{name: "first", verdict: detect.Certain}
	second := &stubWatchdog{name: "second"}

	h := newHarness(t, juryConfig(), banConfig(), first, second)

	h.jury.Attempted(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})
	h.jury.Attempted(detect.Click{Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Equal(t, 1, first.attempted, "a click with no scope is nobody's")
	assert.Equal(t, 1, second.attempted)
	assert.Equal(t, 0, first.seen, "a try the throttle may still refuse is never a verdict")
	assert.Empty(t, h.reports)
}

func TestACallerWithNoScopeIsNotJudged(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Certain}

	h := newHarness(t, juryConfig(), banConfig(), watchdog)

	assert.False(t, h.inspect(detect.Click{Tile: 1, Country: "FR", At: h.clock.Now()}))
	assert.Equal(t, 0, watchdog.seen)
}

func TestTheCallerFactsTravelWithTheBan(t *testing.T) {
	watchdog := &stubWatchdog{name: "sure", verdict: detect.Clear}

	h := newHarness(t, juryConfig(), banConfig(), watchdog)

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
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Certain},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	examination := h.jury.Examine("stranger", "")

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

	h := newHarness(t, juryConfig(), banConfig(), unsure, quiet)

	first := h.clock.Now()
	require.False(t, h.click())
	require.False(t, h.click())
	last := first.Add(time.Second)

	examination := h.jury.Examine("caller", "")

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
	assert.Zero(t, h.flagged())
}

func TestExaminingWithTwoSuspectsReadsGuiltyWithoutBanning(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	require.True(t, h.click())

	examination := h.jury.Examine("caller", "")

	assert.True(t, examination.Guilty)
	assert.Equal(t, 2, examination.Suspects)
	assert.Len(t, h.reports, 1)
}

func TestExaminingAgesASuspicionPastTheWindow(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "first", verdict: detect.Suspect},
		&stubWatchdog{name: "second", verdict: detect.Suspect},
	)

	require.True(t, h.click())
	h.clock.Advance(11 * time.Minute)

	examination := h.jury.Examine("caller", "")

	assert.False(t, examination.Guilty)
	assert.Zero(t, examination.Suspects)
	assert.Equal(t, "clear", examination.Readings[0].Level)
	assert.Equal(t, "first-rule seen=1", examination.Readings[0].Evidence, "the evidence stays to read")
}

func TestAReadingFlappingAcrossABoundRisesOncePerWindow(t *testing.T) {
	watchdog := &stubWatchdog{name: "unsure", verdict: detect.Suspect}

	h := newHarness(t, juryConfig(), banConfig(), watchdog)

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

	h := newHarness(t, juryConfig(), banConfig(), watchdog, quiet)

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
	h := newHarness(t, juryConfig(), banConfig(), sure)

	require.True(t, h.inspect(detect.Click{Scope: "caller", Account: "guest", Tile: 1, Country: "FR", At: h.clock.Now()}))
	require.Len(t, h.reports, 1)
	assert.Equal(t, "guest", h.reports[0].Account)

	sure.verdict = detect.Clear
	h.clock.Advance(time.Second)

	assert.True(t, h.inspect(detect.Click{Scope: "caller", Account: "fresh-cookie", Tile: 1, Country: "FR", At: h.clock.Now()}),
		"the ban fell on the guest's scope too")
	assert.True(t, h.inspect(detect.Click{Scope: "elsewhere", Account: "guest", Tile: 1, Country: "FR", At: h.clock.Now()}),
		"and on the account, wherever it clicks from")
}

func TestASignedInAccountBannedByTheJuryLeavesItsScopeAlone(t *testing.T) {
	sure := &stubWatchdog{name: "sure", verdict: detect.Certain}
	h := newHarness(t, juryConfig(), banConfig(), sure)

	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "bot", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	sure.verdict = detect.Clear
	h.clock.Advance(time.Second)

	assert.False(t, h.inspect(detect.Click{Scope: "campus", Account: "student", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))
}

type banned struct {
	account   string
	alongside string
}

func (h *harness) banned() []banned {
	out := make([]banned, 0, len(h.reports))
	for _, report := range h.reports {
		out = append(out, banned{account: report.Account, alongside: report.Alongside})
	}
	return out
}

func TestAGuiltyVerdictPassesToTheAccomplicesAWatchdogNames(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Suspect, accomplices: []detect.Accomplice{
		{Account: "second", SignedIn: true},
		{Account: "third", SignedIn: true},
	}}
	other := &stubWatchdog{name: "other", verdict: detect.Suspect}
	h := newHarness(t, juryConfig(), banConfig(), ring, other)

	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	assert.Equal(t, []banned{{account: "first"}, {account: "second", alongside: "first"}, {account: "third", alongside: "first"}}, h.banned())
	for _, report := range h.reports {
		assert.Equal(t, 1, report.Offence, report.Account)
		assert.Equal(t, h.clock.Now().Add(time.Hour), report.BannedUntil, report.Account)
	}

	ring.verdict, other.verdict = detect.Clear, detect.Clear
	h.clock.Advance(time.Second)

	assert.True(t, h.inspect(detect.Click{Scope: "elsewhere", Account: "third", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}),
		"an accomplice is dropped wherever it clicks from")
	assert.False(t, h.inspect(detect.Click{Scope: "campus", Account: "student", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}),
		"signed-in accomplices leave their scope alone")
}

func (h *harness) pardon(caller bans.Caller) {
	h.t.Helper()
	_, err := h.banner.Ban(h.t.Context(), caller, time.Hour)
	require.NoError(h.t, err)
	require.NoError(h.t, h.banner.Unban(h.t.Context(), caller))
}

func TestAPardonedCallerBansNobodyNotEvenItsRing(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "ring", verdict: detect.Suspect, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}},
		&stubWatchdog{name: "other", verdict: detect.Suspect},
	)
	h.pardon(bans.Caller{Account: "first"})

	for range 10 {
		require.False(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))
		h.clock.Advance(time.Minute)
	}
	assert.Empty(t, h.reports, "its ring falls with its guilt, and its guilt is pardoned")
	_, ringBanned, err := h.banner.EnforcedSentence(t.Context(), bans.Caller{Account: "second", SignedIn: true})
	require.NoError(t, err)
	assert.False(t, ringBanned)

	h.clock.Advance(pardon)
	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))
	assert.Equal(t, []banned{{account: "first"}, {account: "second", alongside: "first"}}, h.banned())
}

func TestAPardonedAccomplicePassesNoBanOnToItsRing(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "ring", verdict: detect.Suspect, accomplices: []detect.Accomplice{
			{Account: "second", SignedIn: true},
			{Account: "third", SignedIn: true},
		}},
		&stubWatchdog{name: "other", verdict: detect.Suspect},
	)
	h.pardon(bans.Caller{Account: "second"})

	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	assert.Equal(t, []banned{{account: "first"}, {account: "third", alongside: "first"}}, h.banned())
}

func TestAReadingOfTurnsStillNamesTheRingOfAGuiltyCaller(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(),
		&stubWatchdog{name: "turns", verdict: detect.Suspect, turns: true, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}},
		&stubWatchdog{name: "ids", verdict: detect.Suspect},
		&stubWatchdog{name: "cadence", verdict: detect.Suspect},
	)

	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	assert.Equal(t, []banned{{account: "first"}, {account: "second", alongside: "first"}}, h.banned())
}

func TestALoneSuspicionPassesToNoAccomplice(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Suspect, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}}
	h := newHarness(t, juryConfig(), banConfig(), ring, &stubWatchdog{name: "quiet", verdict: detect.Clear})

	require.False(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	assert.Empty(t, h.reports)
	assert.False(t, h.inspect(detect.Click{Scope: "campus", Account: "second", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))
}

func TestAWatchdogThatReadsClearNamesNoAccomplice(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Clear, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}}
	h := newHarness(t, juryConfig(), banConfig(), ring, &stubWatchdog{name: "sure", verdict: detect.Certain})

	require.True(t, h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	assert.Equal(t, []banned{{account: "first"}}, h.banned())
}

func TestAnAccompliceAlreadyFlaggedIsNotReportedAgain(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Certain, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}}
	h := newHarness(t, juryConfig(), banConfig(), ring)

	h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()})
	require.Len(t, h.reports, 2)

	ring.accomplices = []detect.Accomplice{{Account: "first", SignedIn: true}}
	h.clock.Advance(time.Second)
	h.inspect(detect.Click{Scope: "campus", Account: "second", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Len(t, h.reports, 2, "both were flagged a second ago, inside the reflag interval")
}

func TestAGuestAccompliceTakesItsScopeWithIt(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Certain, accomplices: []detect.Accomplice{{Account: "tab"}}}
	h := newHarness(t, juryConfig(), banConfig(), ring)

	require.True(t, h.inspect(detect.Click{Scope: "home", Account: "account", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()}))

	ring.verdict, ring.accomplices = detect.Clear, nil
	h.clock.Advance(time.Second)

	assert.True(t, h.inspect(detect.Click{Scope: "home", Account: "fresh-cookie", Tile: 1, Country: "FR", At: h.clock.Now()}),
		"a guest sheds its account with a new cookie, so its ban falls on its scope too")
}

func TestAFlagThatCannotBeKeptIsReported(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC))
	store := bans.NewMemoryStore()
	cause := errors.New("postgres is down")
	store.FailWith(cause)

	j := jury.New(juryConfig(), bans.NewBook(banConfig(), pardon, clock, store, store, cpatomic.FakeTransactioner{}, &bans.MemoryEnforcer{}),
		convictions.NewMemoryStore(), clock, jury.Hooks{}, &stubWatchdog{name: "sure", verdict: detect.Certain})

	err := j.Inspect(t.Context(), detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})

	require.ErrorIs(t, err, cause)
}

func TestAnAccountIsExaminedOnTheScopeItClickedFromLast(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(), &stubWatchdog{name: "reader", verdict: detect.Suspect})

	h.inspect(detect.Click{Scope: "home", Account: "player", Tile: 1, Country: "FR", At: h.clock.Now()})
	h.clock.Advance(time.Minute)
	h.inspect(detect.Click{Scope: "phone", Account: "player", Tile: 2, Country: "FR", At: h.clock.Now()})

	examination := h.jury.Examine("", "player")

	assert.Equal(t, "phone", examination.Scope)
	assert.Equal(t, "player", examination.Account)
	assert.True(t, examination.Tracked)
	assert.Equal(t, 1, examination.Clicks, "the watchdogs judge the scope, not the account")

	stranger := h.jury.Examine("", "stranger")
	assert.Empty(t, stranger.Scope)
	assert.False(t, stranger.Tracked)
}

func TestEveryBanTheJuryPassesIsKeptAsItIsReported(t *testing.T) {
	ring := &stubWatchdog{name: "ring", verdict: detect.Certain, accomplices: []detect.Accomplice{{Account: "second", SignedIn: true}}}
	h := newHarness(t, juryConfig(), banConfig(), ring)

	h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()})
	h.clock.Advance(time.Second)
	h.inspect(detect.Click{Scope: "campus", Account: "first", SignedIn: true, Tile: 1, Country: "FR", At: h.clock.Now()})

	require.Len(t, h.reports, 2, "the second click is inside the reflag interval")
	at := h.clock.Now().Add(-time.Second)
	assert.Equal(t, []convictions.Conviction{convictions.Of(h.reports[0], at), convictions.Of(h.reports[1], at)},
		h.convictions.Stored())
	assert.Equal(t, "first", h.convictions.Stored()[1].Alongside)
}

func TestAConvictionThatCannotBeKeptIsReportedAndTheBanStillFalls(t *testing.T) {
	h := newHarness(t, juryConfig(), banConfig(), &stubWatchdog{name: "sure", verdict: detect.Certain})
	cause := errors.New("postgres is down")
	h.convictions.FailWith(cause)

	err := h.jury.Inspect(t.Context(), detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})

	require.ErrorIs(t, err, cause)
	assert.Len(t, h.reports, 1, "the log line is still written")
	_, banned, err := h.banner.EnforcedSentence(t.Context(), bans.Caller{Scope: "caller"})
	require.NoError(t, err)
	assert.True(t, banned)
}
