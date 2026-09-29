package antibot_test

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func bound(v float64) *float64 { return &v }

// withTheWeekOfSeptember23 is production's metronome, catcher and defender of 2026-09-23, and the two rules proposed beside them.
func withTheWeekOfSeptember23(config *antibot.Config) {
	config.Metronome.Detector.MaxGap = 7 * time.Second
	config.Metronome.Detector.MaxSpread = 400 * time.Millisecond
	config.Metronome.Detector.Shape.MaxGap = 10 * time.Second
	config.Metronome.Detector.Shape.Clicks = 500
	config.Metronome.Detector.Shape.CertainClicks = 1000
	config.Metronome.Detector.Clock.Period = time.Second
	config.Metronome.Detector.Clock.Clicks = 120
	config.Metronome.Detector.Clock.MinCoherence = bound(0.6)
	config.Metronome.Detector.Clock.CertainClicks = 600
	config.Metronome.Detector.Clock.CertainFor = 30 * time.Minute
	config.Metronome.Detector.Clock.CertainCoherence = bound(0.5)

	config.Catcher.Detector.TrackWindow = time.Hour
	config.Catcher.Detector.Foreign.Window = time.Hour
	config.Catcher.Detector.Foreign.MinClaims = 5
	config.Catcher.Detector.Foreign.CertainClaims = 10

	config.Defender.Enabled = true
	config.Defender.Detector.RetakeWindow = 2 * time.Minute
	config.Defender.Detector.MinClicks = 40
	config.Defender.Detector.MinShare = 0.6
	config.Defender.Detector.TrackWindow = 10 * time.Minute
}

// event is one thing a caller does at one instant: a click, or a claim of somebody else's box.
type event struct {
	at      time.Time
	foreign bool
}

// replay plays events in time order on a random walk over the tiles, and returns when the scope was first dropped.
func (s *stack) replay(scope string, seed uint64, events []event) (time.Time, bool) {
	slices.SortFunc(events, func(a, b event) int { return a.at.Compare(b.at) })

	//nolint:gosec // G404: deterministic PRNG, seeded so the walk replays exactly.
	r := rand.New(rand.NewPCG(seed, seed+1))
	tile := uint32(120000)

	for _, e := range events {
		s.clock.Advance(e.at.Sub(s.clock.Now()))
		if e.foreign {
			s.guard.Foreign(scope)
			continue
		}

		tile = uint32(int(tile) + r.IntN(40) - 20)
		if s.click(scope, tile, "pl") {
			return s.clock.Now(), true
		}
	}

	return time.Time{}, false
}

// hiddenTabClicks is the pl painter from 08:37 on 2026-09-23: bursts on whole seconds, a tick skipped
// now and then, a second click 50-90ms after one in five, and five to seven minutes of refill between.
func hiddenTabClicks(r *rand.Rand, from time.Time, bursts int) []event {
	var events []event

	tick := from.Truncate(time.Second)
	for range bursts {
		for range 55 + r.IntN(15) {
			tick = tick.Add(time.Second)
			if r.IntN(8) == 0 {
				tick = tick.Add(time.Duration(2+r.IntN(2)) * time.Second)
			}
			at := tick.Add(130*time.Millisecond + time.Duration(r.IntN(15))*time.Millisecond)
			events = append(events, event{at: at})
			if r.IntN(5) == 0 {
				events = append(events, event{at: at.Add(time.Duration(50+r.IntN(40)) * time.Millisecond)})
			}
		}
		tick = tick.Add(time.Duration(330+r.IntN(90)) * time.Second)
	}

	return events
}

// poolClaims is the pool's box relay: a claim of another member's box every 20s to 4 minutes, 30 to 80 an hour.
func poolClaims(r *rand.Rand, from time.Time, until time.Time) []event {
	var events []event
	for at := from.Add(time.Duration(r.IntN(60)) * time.Second); at.Before(until); at = at.Add(time.Duration(20+r.IntN(220)) * time.Second) {
		events = append(events, event{at: at, foreign: true})
	}
	return events
}

func TestTheHiddenTabOfSeptember23IsCaughtByItsBeat(t *testing.T) {
	s := newStack(withTheWeekOfSeptember23)
	//nolint:gosec // G404: deterministic PRNG, seeded so the clicks replay exactly.
	r := rand.New(rand.NewPCG(23, 9))

	start := s.clock.Now()
	dropped, ok := s.replay("198.51.100.10", 1, hiddenTabClicks(r, start, 30))

	require.True(t, ok, "every other watchdog read clear for 5h27m in production")
	assert.Less(t, dropped.Sub(start), 90*time.Minute, "the beat held over 600 tries is enough alone")

	verdicts := s.verdicts("198.51.100.10")
	assert.Equal(t, detect.Certain, verdicts["metronome"])
	for _, watchdog := range []string{"retaker", "sequencer", "defender", "catcher", "cohort", "scraper"} {
		assert.Equal(t, detect.Clear, verdicts[watchdog], "%s read clear in production, and still does", watchdog)
	}
	assert.Contains(t, s.reports[0].Opinions[2].String(), "certain clock")
}

func TestTheHiddenTabOfSeptember23IsCaughtSoonerWithTheBoxesItPasses(t *testing.T) {
	s := newStack(withTheWeekOfSeptember23)
	//nolint:gosec // G404: deterministic PRNG, seeded so the clicks replay exactly.
	r := rand.New(rand.NewPCG(23, 9))

	start := s.clock.Now()
	events := hiddenTabClicks(r, start, 30)
	events = append(events, poolClaims(r, start, events[len(events)-1].at)...)
	dropped, ok := s.replay("198.51.100.10", 1, events)

	require.True(t, ok)
	assert.Less(t, dropped.Sub(start), 20*time.Minute, "a beat and a relayed box are two measurements, and two suspicions ban")

	verdicts := s.verdicts("198.51.100.10")
	assert.NotEqual(t, detect.Clear, verdicts["metronome"])
	assert.NotEqual(t, detect.Clear, verdicts["catcher"])
}

// jitteredPainter is the dz clients: a random 0.6-2.1s between clicks and a pause of up to a minute every so often.
func jitteredPainter(r *rand.Rand, from time.Time, until time.Time) []event {
	var events []event
	for at, click := from, 0; at.Before(until); click++ {
		at = at.Add(600*time.Millisecond + time.Duration(r.Int64N(int64(1500*time.Millisecond))))
		if click%45 == 44 {
			at = at.Add(time.Duration(5+r.IntN(55)) * time.Second)
		}
		events = append(events, event{at: at})
	}
	return events
}

func TestTheBoxPoolOfSeptember22IsCaughtOnTheBoxesAlone(t *testing.T) {
	s := newStack(withTheWeekOfSeptember23)
	//nolint:gosec // G404: deterministic PRNG, seeded so the clicks replay exactly.
	r := rand.New(rand.NewPCG(22, 9))

	start := s.clock.Now()
	until := start.Add(2 * time.Hour)
	events := append(jitteredPainter(r, start, until), poolClaims(r, start, until)...)
	dropped, ok := s.replay("2001:db8:e487:f990::/64", 2, events)

	require.True(t, ok)
	assert.Less(t, dropped.Sub(start), 30*time.Minute, "in production it painted for 11h43m before two suspicions met")

	verdicts := s.verdicts("2001:db8:e487:f990::/64")
	assert.Equal(t, detect.Certain, verdicts["catcher"])
	assert.Equal(t, detect.Clear, verdicts["metronome"], "it jitters, and keeps no beat")
}

// heavyHand draws a gap from the quantiles one browser of the heaviest player was measured at on 2026-09-23.
func heavyHand(r *rand.Rand, quantiles [5]time.Duration) time.Duration {
	points := []struct {
		p   float64
		gap time.Duration
	}{
		{0, 25 * time.Millisecond},
		{0.1, quantiles[0]},
		{0.25, quantiles[1]},
		{0.5, quantiles[2]},
		{0.75, quantiles[3]},
		{0.9, quantiles[4]},
		{1, 4 * time.Second},
	}

	u := r.Float64()
	for i := 1; i < len(points); i++ {
		if u <= points[i].p {
			a, b := points[i-1], points[i]
			return a.gap + time.Duration((u-a.p)/(b.p-a.p)*float64(b.gap-a.gap))
		}
	}
	return points[len(points)-1].gap
}

// browser is one of the two on that /64: bursts of the measured gaps, and pauses from seconds up to its longest, 13 minutes.
func browser(r *rand.Rand, from time.Time, until time.Time, quantiles [5]time.Duration) []event {
	var events []event
	for at := from; at.Before(until); {
		for range 20 + r.IntN(100) {
			at = at.Add(heavyHand(r, quantiles))
			events = append(events, event{at: at})
		}
		if r.IntN(12) == 0 {
			at = at.Add(time.Duration(60+r.IntN(720)) * time.Second)
		} else {
			at = at.Add(time.Duration(5+r.IntN(90)) * time.Second)
		}
	}
	return events
}

func TestTheHeavyPlayersOfSeptember23AreNotBanned(t *testing.T) {
	s := newStack(withTheWeekOfSeptember23)
	//nolint:gosec // G404: deterministic PRNG, seeded so the clicks replay exactly.
	r := rand.New(rand.NewPCG(7, 70))

	start := s.clock.Now()
	until := start.Add(3*time.Hour + 39*time.Minute)
	ms := time.Millisecond
	chrome := browser(r, start, until, [5]time.Duration{57 * ms, 273 * ms, 317 * ms, 434 * ms, 691 * ms})
	firefox := browser(r, start.Add(17*time.Second), until, [5]time.Duration{216 * ms, 262 * ms, 297 * ms, 380 * ms, 613 * ms})
	events := slices.Concat(chrome, firefox)

	require.Greater(t, len(events), 8000, "as many clicks as the two browsers tried that morning")
	slices.SortFunc(events, func(a, b event) int { return a.at.Compare(b.at) })

	rules := cpcolls.NewSet[string]()
	for chunk := range slices.Chunk(events, 50) {
		_, dropped := s.replay("2001:db8:861:700::/64", 3, chunk)
		require.False(t, dropped)

		for _, reading := range s.guard.Examine("2001:db8:861:700::/64", "").Readings {
			if reading.Evidence != "" {
				rules.Add(reading.Watchdog + " " + strings.Fields(reading.Evidence)[0])
			}
		}
	}

	assert.Empty(t, s.reports)
	// Two hands merged on one scope already read cadence in production: 2617 of the real clicks did.
	assert.Equal(t, cpcolls.NewSet("metronome cadence"), rules, "no beat, and no box but its own")
}
