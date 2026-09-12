package metronome_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type harness struct {
	watchdog *metronome.Watchdog
	clock    *cptime.FixedClock
	tile     uint32
}

func newHarness(config metronome.Config) *harness {
	h := &harness{
		clock: cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)),
		tile:  1,
	}
	h.watchdog = metronome.New(config, h.clock)
	return h
}

// after waits, then clicks. The tile moves every time so nothing here depends on
// what the map does.
func (h *harness) after(gap time.Duration) (detect.Verdict, detect.Evidence) {
	h.clock.Advance(gap)
	h.tile++

	return h.watchdog.Watch(detect.Click{
		Scope:   "caller",
		Tile:    h.tile,
		Country: "FR",
		At:      h.clock.Now(),
	})
}

func (h *harness) beat(gap time.Duration, count int) (detect.Verdict, detect.Evidence) {
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)

	for range count {
		verdict, evidence = h.after(gap)
	}

	return verdict, evidence
}

func config() metronome.Config {
	return metronome.Config{
		MaxGap:        3 * time.Second,
		MaxSpread:     120 * time.Millisecond,
		MinClicks:     20,
		CertainFor:    10 * time.Minute,
		CertainClicks: 300,
		TrackWindow:   time.Hour,
	}
}

func TestAClockHeldPastAnythingAPersonSustainsIsCertain(t *testing.T) {
	h := newHarness(config())

	verdict, evidence := h.beat(time.Second, 700)

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "cadence", evidence.Rule)
}

func TestAShortSteadyRunIsOnlySuspect(t *testing.T) {
	h := newHarness(config())

	verdict, _ := h.beat(time.Second, 30)

	assert.Equal(t, detect.Suspect, verdict, "half a minute of rhythm is not proof")
}

func TestTempoIsNotTheSignal(t *testing.T) {
	h := newHarness(config())

	// Two and a half seconds between clicks is slower than most players, and it
	// is still a clock. The claim is never that the caller is fast.
	verdict, _ := h.beat(2500*time.Millisecond, 700)

	assert.Equal(t, detect.Certain, verdict)
}

func TestAHandIsNotAClock(t *testing.T) {
	h := newHarness(config())

	gaps := []time.Duration{
		400, 1200, 650, 2100, 380, 900, 1500, 550, 2600, 700,
		1100, 480, 1900, 620, 830, 1400, 520, 2200, 760, 1050,
	}

	var verdict detect.Verdict
	for range 40 {
		for _, gap := range gaps {
			verdict, _ = h.after(gap * time.Millisecond)
		}
	}

	assert.Equal(t, detect.Clear, verdict, "the spread of a hand is the whole defence")
}

func TestLookingAwayOnceEndsTheRun(t *testing.T) {
	h := newHarness(config())

	require.Equal(t, detect.Certain, func() detect.Verdict { v, _ := h.beat(time.Second, 700); return v }())

	// A person stops to look at the map. That is the behaviour a jittered delay
	// cannot imitate cheaply, so the evidence starts again from nothing.
	verdict, _ := h.after(10 * time.Second)
	assert.Equal(t, detect.Clear, verdict)

	verdict, _ = h.beat(time.Second, 18)
	assert.Equal(t, detect.Clear, verdict, "the new run is judged on its own length")

	verdict, _ = h.after(time.Second)
	assert.Equal(t, detect.Suspect, verdict, "and has to earn its way back up from Clear")
}

func TestASmallWobbleIsStillAClock(t *testing.T) {
	h := newHarness(config())

	// Network jitter on a loop sleeping one second. Well inside MaxSpread.
	gaps := []time.Duration{1000, 1012, 995, 1008, 1003, 990, 1015, 1001}

	var verdict detect.Verdict
	for range 90 {
		for _, gap := range gaps {
			verdict, _ = h.after(gap * time.Millisecond)
		}
	}

	assert.Equal(t, detect.Certain, verdict)
}

func TestJitteringWideEnoughBuysTheCallerOut(t *testing.T) {
	h := newHarness(config())

	// Honest about the bound: a bot that randomises its delay by more than
	// MaxSpread is not caught here. It is caught by looking somewhere else,
	// which is the whole reason there is more than one watchdog.
	gaps := []time.Duration{700, 1300, 900, 1500, 600, 1200, 1000, 1400}

	var verdict detect.Verdict
	for range 90 {
		for _, gap := range gaps {
			verdict, _ = h.after(gap * time.Millisecond)
		}
	}

	assert.Equal(t, detect.Clear, verdict)
}
