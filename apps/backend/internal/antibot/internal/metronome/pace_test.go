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

const finalePace = 3

func productionConfig() metronome.Config {
	return metronome.Config{
		MaxGap:        7 * time.Second,
		MaxSpread:     400 * time.Millisecond,
		MinClicks:     120,
		CertainFor:    30 * time.Minute,
		CertainClicks: 900,
		TrackWindow:   15 * time.Minute,
		Stamina: metronome.StaminaConfig{
			Slice:       10 * time.Minute,
			Clicks:      40,
			Window:      6 * time.Hour,
			CertainBusy: 5*time.Hour + 30*time.Minute,
		},
	}
}

type paced struct {
	watchdog *metronome.Watchdog
	clock    *cptime.FixedClock
}

func newPaced(config metronome.Config) *paced {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC))
	return &paced{watchdog: metronome.New(config, clock, func(float64) {}, func(float64) {}, func(time.Duration) {}), clock: clock}
}

func (p *paced) click(gap time.Duration, pace float64) (detect.Verdict, detect.Evidence) {
	p.clock.Advance(gap)
	click := detect.Click{Scope: scope, Account: "ada", Tile: 1, Country: "DZ", At: p.clock.Now(), Pace: pace}
	p.watchdog.Attempted(click)
	return p.watchdog.Watch(click)
}

func (p *paced) worst(d time.Duration, pace float64, gap func() time.Duration) (detect.Verdict, detect.Evidence) {
	var (
		worst    detect.Verdict
		evidence detect.Evidence
	)
	for end := p.clock.Now().Add(d); p.clock.Now().Before(end); {
		if verdict, read := p.click(gap(), pace); verdict >= worst {
			worst, evidence = verdict, read
		}
	}
	return worst, evidence
}

func refillBoundHand(r *rand.Rand, refill time.Duration) func() time.Duration {
	return func() time.Duration {
		return refill + time.Duration(r.IntN(350))*time.Millisecond - 175*time.Millisecond
	}
}

func TestAHandWaitingOnTheFinalesRefillReadsLikeAHandWaitingOnThePlainOne(t *testing.T) {
	refill := 5 * time.Second / finalePace

	unpaced := newPaced(productionConfig())
	verdict, _ := unpaced.worst(40*time.Minute, 1, refillBoundHand(rand.New(rand.NewPCG(31, 10)), refill)) //nolint:gosec // a test draw.
	require.Equal(t, detect.Certain, verdict, "read at its raw gaps, a guest keeping up with the finale's refill would be banned")

	finale := newPaced(productionConfig())
	verdict, evidence := finale.worst(40*time.Minute, finalePace, refillBoundHand(rand.New(rand.NewPCG(31, 10)), refill)) //nolint:gosec // a test draw.

	assert.Equal(t, detect.Clear, verdict, evidence.String())
}

func TestATimerOnTheFinalesRefillIsStillCaught(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 21)) //nolint:gosec // a test draw.
	p := newPaced(productionConfig())

	verdict, evidence := p.worst(35*time.Minute, finalePace, func() time.Duration {
		return 5*time.Second/finalePace + time.Duration(r.IntN(40))*time.Millisecond
	})

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "cadence", evidence.Rule)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "pace", Value: float64(finalePace)})
}

func TestTheLoopOfSeptember14IsStillCaughtAtTheFinalesPace(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 14)) //nolint:gosec // a test draw.
	p := newPaced(productionConfig())

	verdict, evidence := p.worst(35*time.Minute, finalePace, func() time.Duration {
		return 890*time.Millisecond + time.Duration(r.IntN(120))*time.Millisecond
	})

	assert.Equal(t, detect.Certain, verdict, evidence.String())
}

func TestAnEveningAtThePlainPaceAndAFinalePlayedHardIsNotCertain(t *testing.T) {
	r := rand.New(rand.NewPCG(31, 21)) //nolint:gosec // a test draw.
	hand := func() time.Duration { return 6*time.Second + time.Duration(r.IntN(2000))*time.Millisecond }

	unpaced := newPaced(productionConfig())
	_, _ = unpaced.worst(3*time.Hour+40*time.Minute, 1, func() time.Duration { return 12 * time.Second })
	verdict, evidence := unpaced.worst(2*time.Hour, 1, hand)
	require.Equal(t, detect.Certain, verdict, "counted at the plain pace, a finale's easy slices would each be busy")
	require.Equal(t, "stamina", evidence.Rule)

	p := newPaced(productionConfig())

	_, _ = p.worst(3*time.Hour+40*time.Minute, 1, func() time.Duration { return 12 * time.Second })
	verdict, evidence = p.worst(2*time.Hour, finalePace, hand)

	assert.Equal(t, detect.Clear, verdict, evidence.String())
}

func TestALoopAtTheFinalesPaceAfterHoursAtThePlainOneIsCertain(t *testing.T) {
	p := newPaced(productionConfig())

	_, _ = p.worst(4*time.Hour, 1, func() time.Duration { return 5 * time.Second })
	verdict, evidence := p.worst(2*time.Hour, finalePace, func() time.Duration { return 5 * time.Second / finalePace })

	require.Equal(t, detect.Certain, verdict)
	assert.Contains(t, []string{"stamina", "cadence"}, evidence.Rule)
}
