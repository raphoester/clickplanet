package metronome_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Production's bounds, deploy/vps/backend.yaml.
func production() metronome.Config {
	return metronome.Config{
		MaxGap:        7 * time.Second,
		MaxSpread:     400 * time.Millisecond,
		MinClicks:     120,
		CertainFor:    30 * time.Minute,
		CertainClicks: 900,
		Shape:         metronome.ShapeConfig{MaxGap: 10 * time.Second, Clicks: 500, CertainClicks: 1000},
		Stamina: metronome.StaminaConfig{
			Slice: 10 * time.Minute, Clicks: 40, Window: 6 * time.Hour, MinBusy: 5*time.Hour + 30*time.Minute,
		},
		TrackWindow: 15 * time.Minute,
	}
}

const rush = 3

// A guest's refill is one click every 5s, so 5s/3 at three times the pace.
const refillAtRush = 5 * time.Second / rush

type battle struct {
	watchdog *metronome.Watchdog
	clock    *cptime.FixedClock
	tempo    float64
	worst    detect.Verdict
	evidence detect.Evidence
}

var battleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func newBattle(tempo float64, at time.Time) *battle {
	clock := cptime.NewFixedClock(at)
	return &battle{
		watchdog: metronome.New(production(), clock, func(float64) {}, func(float64) {}, func(time.Duration) {}),
		clock:    clock,
		tempo:    tempo,
	}
}

func (b *battle) click(gap time.Duration) {
	b.clock.Advance(gap)
	click := detect.Click{
		Scope: "2001:db8:31:21::/64", Account: "guest", Tile: 1, Country: "DZ", At: b.clock.Now(), Tempo: b.tempo,
	}
	b.watchdog.Attempted(click)
	if verdict, evidence := b.watchdog.Watch(click); verdict > b.worst {
		b.worst, b.evidence = verdict, evidence
	}
}

func (b *battle) play(d time.Duration, gap func() time.Duration) {
	for end := b.clock.Now().Add(d); b.clock.Now().Before(end); {
		b.click(gap())
	}
}

func onTheRefill(r *rand.Rand) func() time.Duration {
	return func() time.Duration {
		return refillAtRush + time.Duration(r.Int64N(int64(300*time.Millisecond)))
	}
}

func TestAPlayerWaitingOnEachRefillAtThreeTimesThePaceIsNotAClock(t *testing.T) {
	b := newBattle(rush, battleStarts)

	b.play(2*time.Hour, onTheRefill(rand.New(rand.NewPCG(31, 10)))) //nolint:gosec // a test draw, not a secret.

	assert.Equal(t, detect.Clear, b.worst, b.evidence.String())
}

func TestTheSamePlayerReadAtThePlainTempoWouldBeCertain(t *testing.T) {
	b := newBattle(0, battleStarts)

	b.play(2*time.Hour, onTheRefill(rand.New(rand.NewPCG(31, 10)))) //nolint:gosec // a test draw, not a secret.

	require.Equal(t, detect.Certain, b.worst, "what the tempo on each click is there to stop")
	assert.Equal(t, "cadence", b.evidence.Rule)
}

func TestATimerOnTheRefillAtThreeTimesThePaceIsStillCertain(t *testing.T) {
	b := newBattle(rush, battleStarts)
	r := rand.New(rand.NewPCG(31, 11)) //nolint:gosec // a test draw, not a secret.

	b.play(30*time.Minute, func() time.Duration {
		return refillAtRush + time.Duration(r.Int64N(int64(20*time.Millisecond)))
	})

	require.Equal(t, detect.Certain, b.worst)
	assert.Equal(t, "cadence", b.evidence.Rule)
}

func TestATimerAtThreeTimesThePaceIsSuspectAsSoonAsAtThePlainPace(t *testing.T) {
	b := newBattle(rush, battleStarts)

	b.play(5*time.Minute, func() time.Duration { return refillAtRush })

	assert.Equal(t, detect.Suspect, b.worst, "120 tries, under four minutes")
}

func TestAnEveningThenTheBattleIsNoBusierThanTheEveningAtThePlainPace(t *testing.T) {
	b := newBattle(1, battleStarts.Add(-3*time.Hour-40*time.Minute))

	b.play(3*time.Hour+40*time.Minute, func() time.Duration { return 10 * time.Second })
	b.tempo = rush
	b.play(2*time.Hour, func() time.Duration { return 6 * time.Second })

	assert.Equal(t, detect.Clear, b.worst, "100 clicks in ten minutes is a third of the battle's pace: not busy")
}

func TestTheBattleAtFullPaceIsStillBusy(t *testing.T) {
	b := newBattle(1, battleStarts.Add(-3*time.Hour-40*time.Minute))

	b.play(3*time.Hour+40*time.Minute, func() time.Duration { return 10 * time.Second })
	b.tempo = rush
	b.play(2*time.Hour, func() time.Duration { return 4 * time.Second })

	require.Equal(t, detect.Suspect, b.worst, "150 clicks in ten minutes at three times the pace is busy")
	assert.Equal(t, "stamina", b.evidence.Rule)
}
