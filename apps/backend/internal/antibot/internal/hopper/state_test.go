package hopper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	north = detect.Point{Z: 1}
	south = detect.Point{Z: -1}
)

func alternate(w *Watchdog, clock *cptime.FixedClock, clicks int) {
	for i := range clicks {
		clock.Advance(time.Second)
		at := north
		if i%2 == 1 {
			at = south
		}
		w.Attempted(detect.Click{Scope: "2001:db8::/64", Account: "player", Position: at, At: clock.Now()})
	}
}

func rules(change func(*Rules)) func() Rules {
	r := Rules{MinAngle: 45, MinGap: 100 * time.Millisecond, MaxGap: 30 * time.Second, MinSteps: 40, CertainSteps: 200}
	change(&r)
	return func() Rules { return r }
}

func asIs(*Rules) {}

func TestStepsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))
	config := Config{}
	reading := rules(func(r *Rules) { r.MinSteps, r.MinShare = 10, 0.5 })

	w := New(config, reading, clock, func(float64) {})
	alternate(w, clock, 11)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, reading, clock, func(float64) {})
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "2001:db8::/64", Account: "player", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestTheFirstTryAfterARestartIsNoStep(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	w := New(Config{}, rules(asIs), clock, func(float64) {})
	alternate(w, clock, 5)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(Config{}, rules(asIs), clock, func(float64) {})
	require.NoError(t, restarted.Load(data))

	clock.Advance(time.Second)
	restarted.Attempted(detect.Click{Scope: "2001:db8::/64", Account: "player", Position: south, At: clock.Now()})

	require.Contains(t, restarted.payers, "account:player")
	assert.Len(t, restarted.payers["account:player"].steps, 4, "the gap across the outage is stitched from two pieces")
}

func TestForgetDropsStepsBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 24 * time.Hour}, rules(asIs), clock, func(float64) {})
	alternate(w, clock, 5)

	w.Forget(clock.Now().Add(time.Second))

	assert.Empty(t, w.payers)
}
