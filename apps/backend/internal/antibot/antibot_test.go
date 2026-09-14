package antibot_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The whole thing built the way internal/planet builds it — through the one
// published constructor, with the bounds cmd/api ships — and driven by callers
// that behave the way the real ones do.
type stack struct {
	guard *antibot.Guard
	clock *cptime.FixedClock

	owner   map[uint32]string
	reports []antibot.Report
	rises   []string
}

func newStack() *stack {
	s := &stack{
		clock: cptime.NewFixedClock(time.Date(2026, 9, 11, 2, 0, 0, 0, time.UTC)),
		owner: map[uint32]string{},
	}

	config := antibot.Config{Enabled: true}

	config.ShadowBan.Enforce = true
	config.ShadowBan.BanDurations = []time.Duration{time.Hour}
	config.ShadowBan.ReflagInterval = 5 * time.Minute

	config.Jury.MinSuspects = 2
	config.Jury.SuspicionWindow = 10 * time.Minute
	config.Jury.TrackWindow = 15 * time.Minute

	config.Retaker.Enabled = true
	config.Retaker.Detector.ReactionWindow = 5 * time.Second
	config.Retaker.Detector.MinReactions = 12
	config.Retaker.Detector.MaxSpread = 120 * time.Millisecond
	config.Retaker.Detector.MaxMedian = 250 * time.Millisecond
	config.Retaker.Detector.TrackWindow = 5 * time.Minute

	config.Sequencer.Enabled = true
	config.Sequencer.Detector.MinSteps = 40
	config.Sequencer.Detector.MinShare = 0.75
	config.Sequencer.Detector.CertainSteps = 200
	config.Sequencer.Detector.CertainShare = 0.95
	config.Sequencer.Detector.TrackWindow = 15 * time.Minute

	config.Metronome.Enabled = true
	config.Metronome.Detector.MaxGap = 3 * time.Second
	config.Metronome.Detector.MaxSpread = 120 * time.Millisecond
	config.Metronome.Detector.MinClicks = 120
	config.Metronome.Detector.CertainFor = 30 * time.Minute
	config.Metronome.Detector.CertainClicks = 900
	config.Metronome.Detector.TrackWindow = 15 * time.Minute

	config.Catcher.Enabled = true
	config.Catcher.Detector.MinCatches = 5
	config.Catcher.Detector.MaxMedian = 3 * time.Second
	config.Catcher.Detector.CertainMedian = 1500 * time.Millisecond
	config.Catcher.Detector.TrackWindow = 30 * time.Minute

	config.Cohort.Enabled = true
	config.Cohort.Detector.StartWindow = 5 * time.Second
	config.Cohort.Detector.MinClicks = 20
	config.Cohort.Detector.MinFlagShare = 0.9
	config.Cohort.Detector.RateRatio = 1.5
	config.Cohort.Detector.LengthRatio = 1.2
	config.Cohort.Detector.QuietAfter = time.Minute
	config.Cohort.Detector.MinMembers = 2
	config.Cohort.Detector.V4Bits = 24
	config.Cohort.Detector.V6Bits = 44
	config.Cohort.Detector.CertainCohorts = 3
	config.Cohort.Detector.CertainMembers = 6
	config.Cohort.Detector.ChainWindow = 30 * time.Minute

	guard, err := antibot.New(config, s.clock, antibot.Observer{
		OnFlag: func(report antibot.Report) { s.reports = append(s.reports, report) },
		OnRise: func(watchdog, level string) { s.rises = append(s.rises, watchdog+" "+level) },
	})
	if err != nil {
		panic(err)
	}

	s.guard = guard

	return s
}

func (s *stack) click(scope string, tile uint32, country string) bool {
	held := s.owner[tile]

	click := antibot.Click{
		Scope:   scope,
		Tile:    tile,
		Country: country,
		At:      s.clock.Now(),
		Held:    held,
		NoOp:    held == country,
	}

	s.guard.Attempted(click)

	drop := s.guard.Inspect(click)
	if !drop {
		s.guard.Committed(click)
		if !click.NoOp {
			s.owner[tile] = country
		}
	}

	return drop
}

func (s *stack) verdicts(scope string) map[string]detect.Verdict {
	out := map[string]detect.Verdict{}
	for _, report := range s.reports {
		if report.Scope != scope {
			continue
		}
		for _, opinion := range report.Opinions {
			out[opinion.Watchdog] = opinion.Verdict
		}
	}
	return out
}

// The caller from the screenshots: a loop over an integer, tuned to sit just
// under the throttle, running all night. It never fights anyone for a tile, so
// the only watchdog this game had before could not see it at all.
func TestTheOvernightSweepIsCaught(t *testing.T) {
	s := newStack()

	var (
		tile    = uint32(180000)
		clicks  int
		dropped bool
	)

	for range 400 {
		s.clock.Advance(time.Second)
		clicks++
		if s.click("sweeper", tile, "FR") {
			dropped = true
			break
		}
		tile++
	}

	require.True(t, dropped)
	require.NotEmpty(t, s.reports)

	// Crossing is what earns its keep here. Neither watchdog is Certain yet —
	// the sequencer wants 200 steps and the metronome wants half an hour — but
	// two of them reading Suspect at once stops the sweep inside three minutes
	// rather than after either bound is reached alone.
	assert.Less(t, clicks, 180, "caught long before either watchdog is sure on its own")

	verdicts := s.verdicts("sweeper")
	assert.Equal(t, detect.Suspect, verdicts["sequencer"])
	assert.Equal(t, detect.Suspect, verdicts["metronome"])
	assert.Equal(t, detect.Clear, verdicts["retaker"], "it never fought anyone, and it did not have to")
}

// The day of 2026-09-14: a bot that jitters its delay reads clear on the
// metronome, so the sequencer's suspicion stands alone and nothing is banned.
// The rise is the signal left of how close it came.
func TestALoneSuspicionIsReportedWithoutABan(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded so the delays are the same every run.
	rng := rand.New(rand.NewPCG(14, 9))
	tile := uint32(180000)

	for range 100 {
		s.clock.Advance(500*time.Millisecond + time.Duration(rng.Int64N(int64(1500*time.Millisecond))))
		require.False(t, s.click("jitterer", tile, "FR"))
		tile++
	}

	assert.Empty(t, s.reports)
	assert.Equal(t, []string{"sequencer suspect"}, s.rises, "once per standing suspicion, worded by the package")
}

// The same bot with the one cheap fix its author would reach for first.
func TestSweepingInARandomOrderStillGetsCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(1, 2))

	var (
		clicks  int
		dropped bool
	)

	for range 3000 {
		s.clock.Advance(time.Second)
		clicks++
		if s.click("shuffler", 180000+uint32(random.IntN(60000)), "FR") {
			dropped = true
			break
		}
	}

	require.True(t, dropped, "shuffling the ids leaves the clock running")

	verdicts := s.verdicts("shuffler")
	assert.Equal(t, detect.Clear, verdicts["sequencer"], "there is no stride left to find")
	assert.Equal(t, detect.Certain, verdicts["metronome"])

	// The honest cost of shuffling: with nothing left to corroborate it, the
	// metronome has to reach Certain on its own, and Certain means certainFor.
	// Half an hour of sweeping is bought for one line of the bot's code. The
	// answer is another watchdog, not a looser bound on this one.
	assert.Greater(t, clicks, 1700, "a lone watchdog has to be sure, and sure takes certainFor")
}

func TestALoopFiringIntoTheThrottleIsCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(9, 10))

	var dropped bool
	for range 4000 {
		s.clock.Advance(950 * time.Millisecond)

		// The throttle refuses about half the tries, unevenly; only those it keeps reach Inspect.
		if random.IntN(2) == 0 {
			s.guard.Attempted(antibot.Click{Scope: "looper", Tile: 1, Country: "BG", At: s.clock.Now()})
			continue
		}

		if s.click("looper", 180000+uint32(random.IntN(60000)), "BG") {
			dropped = true
			break
		}
	}

	require.True(t, dropped)
	assert.Equal(t, detect.Certain, s.verdicts("looper")["metronome"])
}

// A player who is very keen: fast, for a long time, on tiles next to each other
// — and still nothing like either of the above.
func TestAnObsessedPlayerIsNotBanned(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(3, 4))

	tile := uint32(50000)

	for range 90 {
		// A burst of clicks around one area, then a pause to look at the map.
		for range 20 + random.IntN(25) {
			s.clock.Advance(time.Duration(250+random.IntN(1400)) * time.Millisecond)

			tile = uint32(int(tile) + random.IntN(80) - 40)
			require.False(t, s.click("player", tile, "IT"), "a player must never be dropped")
		}

		s.clock.Advance(time.Duration(4+random.IntN(40)) * time.Second)
	}

	assert.Empty(t, s.reports, "nothing about this reads as a machine")
}

// Two players fighting over the same tiles, which is the thing every one of
// these bounds has to survive.
func TestATileWarBansNeither(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(5, 6))

	tile := uint32(70000)

	for range 200 {
		s.clock.Advance(time.Duration(300+random.IntN(1800)) * time.Millisecond)
		require.False(t, s.click("attacker", tile, "IL"))

		s.clock.Advance(time.Duration(300+random.IntN(1800)) * time.Millisecond)
		require.False(t, s.click("defender", tile, "PS"))

		if random.IntN(4) == 0 {
			tile = uint32(int(tile) + random.IntN(60) - 30)
		}
	}

	assert.Empty(t, s.reports)
}

// The reflex bot the first version of this was written for, to prove the move
// into antibot/internal/shadowban did not lose it.
func TestTheReflexBotIsStillCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(7, 8))

	tile := uint32(90000)

	var dropped bool
	for range 40 {
		tile++

		s.clock.Advance(time.Duration(600+random.IntN(2500)) * time.Millisecond)
		s.click("player", tile, "FR")

		// Answers off the update stream, in a band no hand holds.
		s.clock.Advance(time.Duration(70+random.IntN(30)) * time.Millisecond)
		if s.click("reflex", tile, "PS") {
			dropped = true
			break
		}
	}

	require.True(t, dropped)
	assert.Equal(t, detect.Certain, s.verdicts("reflex")["retaker"])
}

// A script that reads the offer off the stream and claims it before the box has
// left its spawn. It paints like a person, so only the boxes give it away.
func TestTheBoxSnatcherIsCaught(t *testing.T) {
	s := newStack()

	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(9, 10))

	tile := uint32(60000)

	var dropped bool
	for box := range 8 {
		// A couple of minutes of ordinary painting between two boxes.
		for range 60 {
			s.clock.Advance(time.Duration(800+random.IntN(2500)) * time.Millisecond)
			tile = uint32(int(tile) + random.IntN(40) - 20)
			if s.click("snatcher", tile, "FR") {
				dropped = true
			}
		}

		if dropped {
			assert.Equal(t, 5, box, "banned on the first click after the fifth box")
			break
		}

		s.guard.Caught("snatcher", time.Duration(200+random.IntN(400))*time.Millisecond)
	}

	require.True(t, dropped)
	assert.Equal(t, detect.Certain, s.verdicts("snatcher")["catcher"])
}

// A good player catches most boxes, some of them fast, and misses the ones that
// went by behind the globe.
func TestAPlayerWhoMissesABoxIsNotBanned(t *testing.T) {
	s := newStack()

	for box := range 30 {
		s.clock.Advance(2 * time.Minute)
		require.False(t, s.click("player", uint32(40000+box), "IT"))

		if box%4 == 3 {
			s.guard.Missed("player")
			continue
		}

		s.guard.Caught("player", 900*time.Millisecond)
	}

	s.clock.Advance(time.Second)
	require.False(t, s.click("player", 50000, "IT"))

	assert.Empty(t, s.reports, "fast, but one box in four got away")
}

// rotation is a pool of identities clicking at once, each on its own scope, from
// its own first click for as long as it stays. Clicks go out in time order, the
// way the server sees them.
type rotation struct {
	scope string
	first time.Time
	stays time.Duration
}

// paint replays the identities at ~30 tiles a minute for flag, on random tiles,
// and returns when each scope was first dropped.
func (s *stack) paint(seed uint64, flag string, identities ...rotation) map[string]time.Time {
	//nolint:gosec // G404: deterministic PRNG, seeded per test so the click
	// stream replays exactly. Not security-relevant.
	random := rand.New(rand.NewPCG(seed, seed+1))

	next := make([]time.Time, len(identities))
	for i, id := range identities {
		next[i] = id.first
	}

	dropped := map[string]time.Time{}

	for {
		who := -1
		for i, id := range identities {
			if next[i].After(id.first.Add(id.stays)) {
				continue
			}
			if who < 0 || next[i].Before(next[who]) {
				who = i
			}
		}
		if who < 0 {
			return dropped
		}

		s.clock.Advance(next[who].Sub(s.clock.Now()))
		scope := identities[who].scope
		if s.click(scope, 100000+uint32(random.IntN(60000)), flag) {
			if _, seen := dropped[scope]; !seen {
				dropped[scope] = s.clock.Now()
			}
		}

		next[who] = next[who].Add(time.Duration(1600+random.IntN(800)) * time.Millisecond)
	}
}

// The pool of 2026-09-14: pairs of /64s out of Firefox's built-in VPN, started
// in the same second, painting bg for ~8 minutes and followed at once by the
// next pair. No identity lives long enough for a watchdog judging one scope, and
// none of them said a word all day.
func TestTheRotatingPoolIsCaught(t *testing.T) {
	s := newStack()

	start := s.clock.Now()
	stays := 476 * time.Second
	pool := []rotation{
		{"2a00:8c40:f0c2:11a0::/64", start, stays},
		{"2a00:8c40:f0c8:5e31::/64", start.Add(1913 * time.Millisecond), stays},
		{"2a00:8c40:f0cd:0b7c::/64", start.Add(3827 * time.Millisecond), stays},
		{"2a00:8c40:f0c7:a1a5::/64", start.Add(192 * time.Second), stays},
		{"2a00:8c40:f0ce:fda7::/64", start.Add(192*time.Second + 640*time.Millisecond), stays},
		{"2a00:8c40:f0c5:6713::/64", start.Add(674 * time.Second), stays},
		{"2a00:8c40:f0c1:9190::/64", start.Add(674*time.Second + 5*time.Millisecond), stays},
	}

	dropped := s.paint(11, "bg", pool...)

	for _, id := range pool[:5] {
		assert.NotContains(t, dropped, id.scope, "one group, then two: nothing yet that two friends could not do")
	}

	for _, id := range pool[5:] {
		require.Contains(t, dropped, id.scope)
		assert.Less(t, dropped[id.scope].Sub(id.first), time.Minute, "the third group is dropped a minute into its eight")

		verdicts := s.verdicts(id.scope)
		assert.Equal(t, detect.Certain, verdicts["cohort"])
		for _, watchdog := range []string{"retaker", "sequencer", "metronome", "catcher"} {
			assert.Equal(t, detect.Clear, verdicts[watchdog], "%s: every other watchdog judges one scope, and saw nothing", watchdog)
		}
	}
}

// Two friends from the same ISP join a bg war in the same second and paint side
// by side for twenty minutes. That is one group, and one group bans nobody.
func TestTwoFriendsJoiningAFlagWarAreNotBanned(t *testing.T) {
	s := newStack()

	start := s.clock.Now()
	dropped := s.paint(12, "bg",
		rotation{"2a00:8c40:f0c5:6713::/64", start, 20 * time.Minute},
		rotation{"2a00:8c40:f0c1:9190::/64", start.Add(400 * time.Millisecond), 20 * time.Minute},
	)

	assert.Empty(t, dropped)
	assert.Empty(t, s.reports)
}

func TestWithTheBlockOffTheGuardPassesEveryClick(t *testing.T) {
	guard, err := antibot.New(antibot.Config{}, nil, antibot.Observer{})
	require.NoError(t, err)

	assert.False(t, guard.Enabled())
	assert.False(t, guard.Inspect(antibot.Click{Scope: "1.2.3.4", Tile: 1, Country: "fr"}))
	assert.False(t, guard.Banned("1.2.3.4"))
}

func TestExaminingABannedScopeCarriesItsSentence(t *testing.T) {
	s := newStack()

	s.clock.Advance(time.Second)
	s.click("player", 1, "FR")
	s.guard.Ban("player", 2*time.Hour)

	examination := s.guard.Examine("player")

	assert.True(t, examination.Tracked)
	assert.True(t, examination.Banned)
	assert.Equal(t, 1, examination.Offence)
	assert.Equal(t, s.clock.Now().Add(2*time.Hour), examination.BannedUntil)
	watchdogs := make([]string, 0, len(examination.Readings))
	for _, reading := range examination.Readings {
		watchdogs = append(watchdogs, reading.Watchdog)
	}
	assert.Equal(t, []string{"retaker", "sequencer", "metronome", "catcher", "cohort"}, watchdogs)
	assert.False(t, examination.Guilty)
}

func TestAGuardThatIsOffExaminesNothing(t *testing.T) {
	guard, err := antibot.New(antibot.Config{}, nil, antibot.Observer{})
	require.NoError(t, err)

	assert.Equal(t, antibot.Examination{Scope: "player"}, guard.Examine("player"))
}

func TestValidateNamesTheCohortBoundItRefuses(t *testing.T) {
	config := antibot.Config{Enabled: true}
	config.Cohort.Enabled = true
	config.Cohort.Detector.MinMembers = 1

	err := config.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "antiBot.cohort.detector: minMembers")

	config.Cohort.Enabled = false
	assert.NoError(t, config.Validate(), "a watchdog that is off has no bounds to get wrong")
}
