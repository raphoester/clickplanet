package antibot_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
)

// The whole thing wired the way internal/clicks wires it, driven by callers that
// behave the way the real ones do. The bounds here are the ones cmd/api ships.
type stack struct {
	jury  *antibot.Jury
	clock *fakeClock

	owner   map[uint32]string
	reports []antibot.Report
}

func newStack() *stack {
	s := &stack{
		clock: &fakeClock{now: time.Date(2026, 9, 11, 2, 0, 0, 0, time.UTC)},
		owner: map[uint32]string{},
	}

	banner := shadowban.New(shadowban.Config{
		Enforce:        true,
		BanDuration:    time.Hour,
		ReflagInterval: 5 * time.Minute,
	}, s.clock)

	s.jury = antibot.NewJury(
		antibot.Config{MinSuspects: 2, SuspicionWindow: 10 * time.Minute, TrackWindow: 15 * time.Minute},
		banner,
		s.clock,
		func(report antibot.Report) { s.reports = append(s.reports, report) },
		retaker.New(retaker.Config{
			ReactionWindow: 5 * time.Second,
			MinReactions:   12,
			MaxSpread:      120 * time.Millisecond,
			MaxMedian:      250 * time.Millisecond,
			TrackWindow:    5 * time.Minute,
		}, s.clock, nil),
		sequencer.New(sequencer.Config{
			MinSteps:     40,
			MinShare:     0.75,
			CertainSteps: 200,
			CertainShare: 0.95,
			TrackWindow:  15 * time.Minute,
		}, s.clock),
		metronome.New(metronome.Config{
			MaxGap:        3 * time.Second,
			MaxSpread:     120 * time.Millisecond,
			MinClicks:     120,
			CertainFor:    30 * time.Minute,
			CertainClicks: 900,
			TrackWindow:   15 * time.Minute,
		}, s.clock),
	)

	return s
}

func (s *stack) click(scope string, tile uint32, country string) bool {
	held := s.owner[tile]

	click := antibot.Click{
		Scope:   scope,
		Tile:    tile,
		Country: country,
		At:      s.clock.now,
		Held:    held,
		NoOp:    held == country,
	}

	drop := s.jury.Inspect(click)
	if !drop {
		s.jury.Committed(click)
		if !click.NoOp {
			s.owner[tile] = country
		}
	}

	return drop
}

func (s *stack) advance(d time.Duration) { s.clock.now = s.clock.now.Add(d) }

func (s *stack) verdicts(scope string) map[string]antibot.Verdict {
	out := map[string]antibot.Verdict{}
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
		s.advance(time.Second)
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
	assert.Equal(t, antibot.Suspect, verdicts["sequencer"])
	assert.Equal(t, antibot.Suspect, verdicts["metronome"])
	assert.Equal(t, antibot.Clear, verdicts["retaker"], "it never fought anyone, and it did not have to")
}

// The same bot with the one cheap fix its author would reach for first.
func TestSweepingInARandomOrderStillGetsCaught(t *testing.T) {
	s := newStack()

	random := rand.New(rand.NewPCG(1, 2))

	var (
		clicks  int
		dropped bool
	)

	for range 3000 {
		s.advance(time.Second)
		clicks++
		if s.click("shuffler", 180000+uint32(random.IntN(60000)), "FR") {
			dropped = true
			break
		}
	}

	require.True(t, dropped, "shuffling the ids leaves the clock running")

	verdicts := s.verdicts("shuffler")
	assert.Equal(t, antibot.Clear, verdicts["sequencer"], "there is no stride left to find")
	assert.Equal(t, antibot.Certain, verdicts["metronome"])

	// The honest cost of shuffling: with nothing left to corroborate it, the
	// metronome has to reach Certain on its own, and Certain means certainFor.
	// Half an hour of sweeping is bought for one line of the bot's code. The
	// answer is another watchdog, not a looser bound on this one.
	assert.Greater(t, clicks, 1700, "a lone watchdog has to be sure, and sure takes certainFor")
}

// A player who is very keen: fast, for a long time, on tiles next to each other
// — and still nothing like either of the above.
func TestAnObsessedPlayerIsNotBanned(t *testing.T) {
	s := newStack()

	random := rand.New(rand.NewPCG(3, 4))

	tile := uint32(50000)

	for range 90 {
		// A burst of clicks around one area, then a pause to look at the map.
		for range 20 + random.IntN(25) {
			s.advance(time.Duration(250+random.IntN(1400)) * time.Millisecond)

			tile = uint32(int(tile) + random.IntN(80) - 40)
			require.False(t, s.click("player", tile, "IT"), "a player must never be dropped")
		}

		s.advance(time.Duration(4+random.IntN(40)) * time.Second)
	}

	assert.Empty(t, s.reports, "nothing about this reads as a machine")
}

// Two players fighting over the same tiles, which is the thing every one of
// these bounds has to survive.
func TestATileWarBansNeither(t *testing.T) {
	s := newStack()

	random := rand.New(rand.NewPCG(5, 6))

	tile := uint32(70000)

	for range 200 {
		s.advance(time.Duration(300+random.IntN(1800)) * time.Millisecond)
		require.False(t, s.click("attacker", tile, "IL"))

		s.advance(time.Duration(300+random.IntN(1800)) * time.Millisecond)
		require.False(t, s.click("defender", tile, "PS"))

		if random.IntN(4) == 0 {
			tile = uint32(int(tile) + random.IntN(60) - 30)
		}
	}

	assert.Empty(t, s.reports)
}

// The reflex bot the first version of this was written for, to prove the move
// out of kernel/shadowban did not lose it.
func TestTheReflexBotIsStillCaught(t *testing.T) {
	s := newStack()

	random := rand.New(rand.NewPCG(7, 8))

	tile := uint32(90000)

	var dropped bool
	for range 40 {
		tile++

		s.advance(time.Duration(600+random.IntN(2500)) * time.Millisecond)
		s.click("player", tile, "FR")

		// Answers off the update stream, in a band no hand holds.
		s.advance(time.Duration(70+random.IntN(30)) * time.Millisecond)
		if s.click("reflex", tile, "PS") {
			dropped = true
			break
		}
	}

	require.True(t, dropped)
	assert.Equal(t, antibot.Certain, s.verdicts("reflex")["retaker"])
}
