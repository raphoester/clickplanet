package metronome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var restartConfig = Config{MaxGap: 3 * time.Second, MinClicks: 60, CertainFor: 10 * time.Minute, CertainClicks: 60, TrackWindow: time.Hour}

var looper = detect.Click{Scope: "loop"}.Payer()

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

	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	require.NoError(t, restarted.Load(data))
	restarted.Resume(detect.Outage{From: savedAt, To: clock.Now()})

	return restarted
}

func TestARunSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
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

	w := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
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
	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
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

type savedByScope struct {
	Callers  []savedCallerByScope
	Slice    int64
	Spenders []savedSpender
}

type savedCallerByScope struct {
	Scope     string
	LastSeen  int64
	RunStart  int64
	RunClicks int
	Gaps      []int64
	Shape     []int64
	Tries     []int64
}

func TestASectionSavedByScopeKeepsItsStaminaAndNoRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))
	config := Config{Stamina: StaminaConfig{MinBusy: 10 * time.Minute, Clicks: 1}}

	w := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	spender := detect.Click{Scope: "home", Account: "hand", At: clock.Now()}

	data, err := evidence.Encode(savedByScope{
		Callers: []savedCallerByScope{{
			Scope:     "home",
			LastSeen:  evidence.Nanos(clock.Now()),
			RunStart:  evidence.Nanos(clock.Now().Add(-time.Hour)),
			RunClicks: 3600,
			Gaps:      []int64{int64(time.Second), int64(time.Second)},
		}},
		Slice: int64(w.config.Stamina.Slice),
		Spenders: []savedSpender{{
			Payer:    spender.Payer(),
			LastSeen: evidence.Nanos(clock.Now()),
			Slices:   []savedSlice{{Index: w.sliceOf(clock.Now()), Clicks: 40}},
		}},
	})
	require.NoError(t, err)

	require.NoError(t, w.Load(data), "a section from before the payer must not cost the boot")

	assert.Empty(t, w.payers, "a run timed on a whole scope is nobody's run")

	verdict, read := w.Watch(spender)
	assert.Equal(t, detect.Suspect, verdict, "the busy time was the payer's already")
	assert.Equal(t, "stamina", read.Rule)
}

func TestTheShapeSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	config := Config{TrackWindow: time.Hour, Shape: ShapeConfig{Clicks: 20, CertainClicks: 40}}

	w := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	for i := range 60 {
		clock.Advance(time.Duration(600+i%16*100) * time.Millisecond)
		w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	require.NoError(t, restarted.Load(data))

	require.NotEmpty(t, w.payers[looper].shape)
	assert.Equal(t, w.payers[looper].shape, restarted.payers[looper].shape)
}

func TestStaminaSurvivesASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 21, 51, 0, 0, time.UTC))
	config := Config{Stamina: StaminaConfig{CertainBusy: 5*time.Hour + 30*time.Minute}}

	w := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	for range 6 * 60 * 12 {
		clock.Advance(5 * time.Second)
		w.Watch(detect.Click{Scope: "night", Account: "night", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
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

	w := New(Config{}, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	w.Watch(detect.Click{Scope: "night", Account: "night", At: clock.Now()})

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(Config{Stamina: StaminaConfig{Slice: 5 * time.Minute}}, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	require.NoError(t, restarted.Load(data))

	assert.Empty(t, restarted.spenders)
}

func TestARestartIsNeitherABreakNorABurst(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	for range 3 {
		loop(w, clock, 4*time.Minute)
		w = restart(t, w, clock, 40*time.Second)
	}
	loop(w, clock, 2*time.Second)

	p := w.payers[looper]
	require.NotNil(t, p)

	assert.Equal(t, 3*4*60+2, p.runClicks, "three restarts, one run")
	assert.Equal(t, 12*time.Minute+time.Second, p.ran, "the outages are not time the loop was watched")
	for _, gap := range p.gaps {
		assert.Equal(t, time.Second, gap, "a stitched gap is never a sample")
	}

	verdict, _ := w.Watch(detect.Click{Scope: "loop", At: clock.Now()})
	assert.Equal(t, detect.Certain, verdict)
}

func TestAPauseAroundARestartStillEndsTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	loop(w, clock, 2*time.Minute)

	clock.Advance(2 * time.Second)
	w = restart(t, w, clock, 40*time.Second)
	clock.Advance(2 * time.Second)
	w.Attempted(detect.Click{Scope: "loop", At: clock.Now()})

	assert.Equal(t, 1, w.payers[looper].runClicks)
}

func TestWithoutAnOutageTheRestartGapBreaksTheRun(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	loop(w, clock, 2*time.Minute)

	data, err := w.Save()
	require.NoError(t, err)
	clock.Advance(40 * time.Second)

	restarted := New(restartConfig, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	require.NoError(t, restarted.Load(data))
	loop(restarted, clock, time.Second)

	assert.Equal(t, 1, restarted.payers[looper].runClicks)
}

func TestASliceKeepsTheTempoItSawAcrossARestart(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC))
	config := Config{Stamina: StaminaConfig{MinBusy: 10 * time.Minute, Clicks: 2}}
	click := detect.Click{Scope: "home", Account: "player", At: clock.Now(), Tempo: 3}

	w := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	for range 3 {
		w.Watch(click)
	}

	data, err := w.Save()
	require.NoError(t, err)
	restarted := New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {})
	require.NoError(t, restarted.Load(data))

	click.Tempo = 1
	verdict, evidence := restarted.Watch(click)
	assert.Equal(t, detect.Clear, verdict, "four clicks in a slice that saw three times the pace: under six, not busy. %s", evidence)
}
