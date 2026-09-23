package metronome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var restartConfig = Config{MaxGap: 3 * time.Second, MinClicks: 60, CertainFor: 10 * time.Minute, CertainClicks: 60, TrackWindow: time.Hour}

// loop tries a click every second for d.
func loop(w *Watchdog, clock *cptime.FixedClock, d time.Duration) {
	for end := clock.Now().Add(d); clock.Now().Before(end); {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "loop", Tile: 1, Country: "BG", At: clock.Now()})
	}
}

// restart saves w, lets the outage pass, and hands back the process that loaded it.
func restart(t *testing.T, w *Watchdog, clock *cptime.FixedClock, outage time.Duration) *Watchdog {
	t.Helper()

	savedAt := clock.Now()
	data, err := w.Save()
	require.NoError(t, err)

	clock.Advance(outage)

	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {})
	require.NoError(t, restarted.Load(data))
	restarted.Resume(detect.Outage{From: savedAt, To: clock.Now()})

	return restarted
}

func TestARunSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {})
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {})
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "loop", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestTheShapeSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	config := Config{TrackWindow: time.Hour, Shape: ShapeConfig{Clicks: 20, CertainClicks: 40}}

	w := New(config, clock, func(float64) {}, func(float64) {})
	for i := range 60 {
		clock.Advance(time.Duration(600+i%16*100) * time.Millisecond)
		w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(config, clock, func(float64) {}, func(float64) {})
	require.NoError(t, restarted.Load(data))

	assert.Equal(t, w.callers["loop"].shape, restarted.callers["loop"].shape)
}

func TestARestartIsNeitherABreakNorABurst(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {})
	for range 3 {
		loop(w, clock, 4*time.Minute)
		w = restart(t, w, clock, 40*time.Second)
	}
	loop(w, clock, 2*time.Second)

	c := w.callers["loop"]
	require.NotNil(t, c)

	assert.Equal(t, 3*4*60+2, c.runClicks, "three restarts, one run")
	assert.Equal(t, 12*time.Minute+time.Second, c.lastSeen.Sub(c.runStart), "the outages are not time the loop was watched")
	for _, gap := range c.gaps {
		assert.Equal(t, time.Second, gap, "a stitched gap is never a sample")
	}

	verdict, _ := w.Watch(detect.Click{Scope: "loop", At: clock.Now()})
	assert.Equal(t, detect.Certain, verdict)
}

func TestAPauseAroundARestartStillEndsTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {})
	loop(w, clock, 2*time.Minute)

	// Stopped 2s before the save and came back 2s after the new process started watching.
	clock.Advance(2 * time.Second)
	w = restart(t, w, clock, 40*time.Second)
	clock.Advance(2 * time.Second)
	w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})

	assert.Equal(t, 1, w.callers["loop"].runClicks)
}

func TestWithoutAnOutageTheRestartGapBreaksTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {})
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	clock.Advance(40 * time.Second)

	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {})
	require.NoError(t, restarted.Load(data))
	loop(restarted, clock, time.Second)

	assert.Equal(t, 1, restarted.callers["loop"].runClicks)
}
