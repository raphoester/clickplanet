package metronome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The bounds the watchdog shipped with, when it filled the ones a config left out.
func bounds(change func(*Rules)) Rules {
	r := Rules{
		MaxGap:        3 * time.Second,
		MaxSpread:     120 * time.Millisecond,
		MinClicks:     120,
		CertainFor:    30 * time.Minute,
		CertainClicks: 900,
		Shape:         ShapeRules{MaxGap: 10 * time.Second, Clicks: 500, CertainClicks: 1000},
		Clock:         ClockRules{Period: time.Second, Clicks: 120, CertainClicks: 600, CertainFor: 30 * time.Minute},
		Stamina:       StaminaRules{Clicks: 40, Window: 6 * time.Hour},
	}
	change(&r)
	return r
}

func asIs(*Rules) {}

func newWatchdog(config Config, rules Rules, clock cptime.Clock) *Watchdog {
	return New(config, func() Rules { return rules }, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
}

var (
	restartConfig = Config{TrackWindow: time.Hour}
	restartRules  = bounds(func(r *Rules) { r.MinClicks, r.CertainFor, r.CertainClicks = 60, 10*time.Minute, 60 })
)

func loop(w *Watchdog, clock *cptime.FixedClock, d time.Duration) {
	for end := clock.Now().Add(d); clock.Now().Before(end); {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "loop", Tile: 1, Country: "BG", At: clock.Now()})
	}
}

func restart(t *testing.T, w *Watchdog, clock *cptime.FixedClock, outage time.Duration) *Watchdog {
	t.Helper()

	savedAt := clock.Now()
	data, err := w.Save()
	require.NoError(t, err)

	clock.Advance(outage)

	restarted := newWatchdog(restartConfig, restartRules, clock)
	require.NoError(t, restarted.Load(data))
	restarted.Resume(detect.Outage{From: savedAt, To: clock.Now()})

	return restarted
}

func TestARunSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(restartConfig, restartRules, clock)
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(restartConfig, restartRules, clock)
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "loop", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestRunsBehindOneScopeAreSavedApart(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(restartConfig, restartRules, clock)
	early := []time.Duration{100, 700, 250, 900, 400}
	for i := range 120 {
		wait := early[i%len(early)] * time.Millisecond

		clock.Advance(wait)
		w.Attempted(detect.Click{Scope: "home", Account: "hand", At: clock.Now()})
		clock.Advance(time.Second - wait)
		w.Attempted(detect.Click{Scope: "home", Account: "loop", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(restartConfig, restartRules, clock)
	require.NoError(t, restarted.Load(data))

	for account, want := range map[string]detect.Verdict{"loop": detect.Suspect, "hand": detect.Clear} {
		next := detect.Click{Scope: "home", Account: account, At: clock.Now()}
		wantVerdict, wantEvidence := w.Watch(next)
		gotVerdict, gotEvidence := restarted.Watch(next)

		require.Equal(t, want, wantVerdict, account)
		assert.Equal(t, wantVerdict, gotVerdict, account)
		assert.Equal(t, wantEvidence, gotEvidence, account)
	}
}

func TestTheShapeSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	config := Config{TrackWindow: time.Hour}
	maxSkew, certainSkew := 0.25, 0.15
	shorter := bounds(func(r *Rules) {
		r.Shape.Clicks, r.Shape.CertainClicks, r.Shape.MaxSkew, r.Shape.CertainSkew = 20, 40, &maxSkew, &certainSkew
	})

	w := newWatchdog(config, shorter, clock)
	for i := range 60 {
		clock.Advance(time.Duration(600+i%16*100) * time.Millisecond)
		w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(config, shorter, clock)
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "loop", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Certain, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestStaminaSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 21, 51, 0, 0, time.UTC))
	sure := bounds(func(r *Rules) { r.Stamina.CertainBusy = 5*time.Hour + 30*time.Minute })

	w := newWatchdog(Config{}, sure, clock)
	for range 6 * 60 * 12 {
		clock.Advance(5 * time.Second)
		w.Watch(detect.Click{Scope: "night", Account: "night", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(Config{}, sure, clock)
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "night", Account: "night", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Certain, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestStaminaCountedInAnotherSliceIsDropped(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 21, 51, 0, 0, time.UTC))

	w := newWatchdog(Config{}, bounds(asIs), clock)
	w.Watch(detect.Click{Scope: "night", Account: "night", At: clock.Now()})

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(Config{Stamina: StaminaConfig{Slice: 5 * time.Minute}}, bounds(asIs), clock)
	require.NoError(t, restarted.Load(data))

	loaded, err := restarted.Save()
	require.NoError(t, err)
	empty, err := newWatchdog(Config{Stamina: StaminaConfig{Slice: 5 * time.Minute}}, bounds(asIs), clock).Save()
	require.NoError(t, err)
	assert.Equal(t, empty, loaded)
}

func TestARestartIsNeitherABreakNorABurst(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(restartConfig, restartRules, clock)
	for range 3 {
		loop(w, clock, 4*time.Minute)
		w = restart(t, w, clock, 40*time.Second)
	}
	loop(w, clock, 2*time.Second)

	verdict, read := w.Watch(detect.Click{Scope: "loop", At: clock.Now()})
	assert.Equal(t, detect.Certain, verdict)
	assert.Contains(t, read.Fields, detect.Field{Key: "clicks", Value: 3*4*60 + 2}, "three restarts, one run")
	assert.Contains(t, read.Fields, detect.Field{Key: "sustained", Value: 12*time.Minute + time.Second},
		"the outages are not time the loop was watched")
	assert.Contains(t, read.Fields, detect.Field{Key: "spread", Value: time.Duration(0)}, "a stitched gap is never a sample")
}

func TestAPauseAroundARestartStillEndsTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(restartConfig, restartRules, clock)
	loop(w, clock, 2*time.Minute)

	clock.Advance(2 * time.Second)
	w = restart(t, w, clock, 40*time.Second)
	clock.Advance(2 * time.Second)
	w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})
	loop(w, clock, time.Minute)

	_, read := w.Watch(detect.Click{Scope: "loop", At: clock.Now()})
	assert.Contains(t, read.Fields, detect.Field{Key: "clicks", Value: 61}, "the run started again after the restart")
}

func TestWithoutAnOutageTheRestartGapBreaksTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(restartConfig, restartRules, clock)
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	clock.Advance(40 * time.Second)

	restarted := newWatchdog(restartConfig, restartRules, clock)
	require.NoError(t, restarted.Load(data))
	loop(restarted, clock, time.Second)
	loop(restarted, clock, time.Minute)

	_, read := restarted.Watch(detect.Click{Scope: "loop", At: clock.Now()})
	assert.Contains(t, read.Fields, detect.Field{Key: "clicks", Value: 61}, "the run started again after the restart")
}

func TestASliceKeepsTheTempoItSawAcrossARestart(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC))
	rules := bounds(func(r *Rules) { r.Stamina = StaminaRules{Clicks: 2, Window: 6 * time.Hour, MinBusy: 10 * time.Minute} })
	click := detect.Click{Scope: "home", Account: "player", At: clock.Now(), Tempo: 3}

	w := newWatchdog(Config{}, rules, clock)
	for range 3 {
		w.Watch(click)
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := newWatchdog(Config{}, rules, clock)
	require.NoError(t, restarted.Load(data))

	click.Tempo = 1
	verdict, evidence := restarted.Watch(click)
	assert.Equal(t, detect.Clear, verdict, "four clicks in a slice that saw three times the pace: under six, not busy. %s", evidence)
}
