package metronome_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
)

func coherence(v float64) *float64 { return &v }

// clockConfig is production's metronome of 2026-09-23 with the clock bounds proposed beside it.
func clockConfig() metronome.Config {
	return metronome.Config{
		MaxGap:        7 * time.Second,
		MaxSpread:     400 * time.Millisecond,
		MinClicks:     120,
		CertainFor:    30 * time.Minute,
		CertainClicks: 900,
		TrackWindow:   15 * time.Minute,
		Shape:         metronome.ShapeConfig{MaxGap: 10 * time.Second, Clicks: 500, CertainClicks: 1000},
		Clock: metronome.ClockConfig{
			Period:           time.Second,
			Clicks:           120,
			MinCoherence:     coherence(0.6),
			CertainClicks:    600,
			CertainFor:       30 * time.Minute,
			CertainCoherence: coherence(0.5),
		},
	}
}

// at advances to an instant and tries a click there.
func (h *harness) at(instant time.Time) (detect.Verdict, detect.Evidence) {
	return h.after(instant.Sub(h.clock.Now()))
}

// hiddenTab is the pl painter of 2026-09-23: a script in a background tab, whose timers the browser
// fires on whole seconds. It spends the bucket in a burst on the beat, skipping a tick now and then
// and sending a second click 70ms after some, then waits whole minutes for the refill.
func hiddenTab(h *harness, r *rand.Rand, bursts int) (detect.Verdict, detect.Evidence) {
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)

	tick := h.clock.Now().Truncate(time.Second).Add(time.Second)
	for range bursts {
		for range 55 + r.IntN(15) {
			tick = tick.Add(time.Duration(1+boolInt(r.IntN(8) == 0)*(2+r.IntN(2))) * time.Second)
			verdict, evidence = h.at(tick.Add(135*time.Millisecond + time.Duration(r.IntN(10))*time.Millisecond))
			if r.IntN(5) == 0 {
				verdict, evidence = h.after(70 * time.Millisecond)
			}
		}
		tick = tick.Add(time.Duration(330+r.IntN(90)) * time.Second)
	}

	return verdict, evidence
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// aboutASecond is a hand clicking roughly once a second: the error of each click adds to the next.
func aboutASecond(r *rand.Rand, click int) time.Duration {
	if click%80 == 79 {
		return time.Duration(20+r.IntN(300)) * time.Second
	}
	return time.Second + time.Duration(r.NormFloat64()*float64(120*time.Millisecond))
}

func TestAHiddenTabIsOnTheBeatWithinTwoBursts(t *testing.T) {
	h := newHarness(clockConfig())
	r := rand.New(rand.NewPCG(23, 9)) //nolint:gosec // a test draw, not a secret.

	verdict, evidence := hiddenTab(h, r, 2)

	require.Equal(t, detect.Suspect, verdict)
	assert.Equal(t, "clock", evidence.Rule)
}

func TestAHiddenTabHeldForAnHourIsCertain(t *testing.T) {
	h := newHarness(clockConfig())
	r := rand.New(rand.NewPCG(23, 9)) //nolint:gosec // a test draw, not a secret.

	verdict, evidence := hiddenTab(h, r, 9)

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "clock", evidence.Rule)
	assert.Contains(t, evidence.String(), "period=1s")
}

func TestTheBeatIsNotCertainBeforeItHasLastedHalfAnHour(t *testing.T) {
	h := newHarness(clockConfig())

	var verdict detect.Verdict
	for range 700 {
		verdict, _ = h.after(2 * time.Second)
	}

	require.Less(t, 700*2*time.Second, 30*time.Minute)
	assert.Equal(t, detect.Suspect, verdict, "a song lasts minutes; a clock outlasts it")
}

func TestAHandClickingAboutOnceASecondKeepsNoBeat(t *testing.T) {
	h := newHarness(clockConfig())

	verdict, _ := play(h, 3000, aboutASecond)

	assert.Equal(t, detect.Clear, verdict)
}

func TestAHandLeaningLongKeepsNoBeat(t *testing.T) {
	h := newHarness(clockConfig())

	verdict, _ := play(h, 3000, hand)

	assert.Equal(t, detect.Clear, verdict)
}

func TestAClockWithNoBoundOnlyMeasures(t *testing.T) {
	c := clockConfig()
	c.Clock.MinCoherence = nil
	c.Clock.CertainCoherence = nil
	h := newHarness(c)
	r := rand.New(rand.NewPCG(23, 9)) //nolint:gosec // a test draw, not a secret.

	verdict, _ := hiddenTab(h, r, 9)

	assert.Equal(t, detect.Clear, verdict)
}

func TestTheBeatSurvivesARestart(t *testing.T) {
	h := newHarness(clockConfig())
	r := rand.New(rand.NewPCG(23, 9)) //nolint:gosec // a test draw, not a secret.
	_, _ = hiddenTab(h, r, 9)

	saved, err := h.watchdog.Save()
	require.NoError(t, err)

	restarted := newHarness(clockConfig())
	restarted.clock = h.clock
	require.NoError(t, restarted.watchdog.Load(saved))

	verdict, evidence := restarted.at(h.clock.Now().Truncate(time.Second).Add(20*time.Second + 138*time.Millisecond))

	assert.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "clock", evidence.Rule)
}
