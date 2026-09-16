package metronome_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
)

func skew(v float64) *float64 { return &v }

func shapeConfig() metronome.Config {
	c := config()
	c.Shape = metronome.ShapeConfig{
		MaxGap:        10 * time.Second,
		Clicks:        500,
		MaxSkew:       skew(0.25),
		CertainClicks: 1000,
		CertainSkew:   skew(0.15),
	}
	return c
}

// randomSleep is the bot of 2026-09-16: a random sleep between 0.6s and 2.1s, and a pause every so often.
func randomSleep(r *rand.Rand, click int) time.Duration {
	if click%40 == 39 {
		return time.Duration(5+r.IntN(35)) * time.Second
	}
	return 600*time.Millisecond + time.Duration(r.Int64N(int64(1500*time.Millisecond)))
}

// hand leans long: mostly quick clicks, now and then a slow one, as players measured the same day did.
func hand(r *rand.Rand, click int) time.Duration {
	if click%25 == 24 {
		return time.Duration(12+r.IntN(40)) * time.Second
	}
	return time.Duration(math.Exp(r.NormFloat64()*0.8-0.9) * float64(time.Second))
}

func play(h *harness, clicks int, gap func(*rand.Rand, int) time.Duration) (detect.Verdict, detect.Evidence) {
	r := rand.New(rand.NewPCG(16, 9)) //nolint:gosec // a test draw, not a secret.

	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)
	for click := range clicks {
		verdict, evidence = h.after(gap(r, click))
	}
	return verdict, evidence
}

func TestARandomSleepIsStillAClock(t *testing.T) {
	h := newHarness(shapeConfig())

	verdict, evidence := play(h, 600, randomSleep)

	require.Equal(t, detect.Suspect, verdict)
	assert.Equal(t, "shape", evidence.Rule)
}

func TestARandomSleepHeldLongIsCertain(t *testing.T) {
	h := newHarness(shapeConfig())

	verdict, evidence := play(h, 1100, randomSleep)

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "shape", evidence.Rule)
}

func TestAHandLeansLong(t *testing.T) {
	h := newHarness(shapeConfig())

	verdict, _ := play(h, 3000, hand)

	assert.Equal(t, detect.Clear, verdict)
}

func TestTheShapeIsNotJudgedBeforeItsWindowFills(t *testing.T) {
	h := newHarness(shapeConfig())

	verdict, _ := play(h, 400, randomSleep)

	assert.Equal(t, detect.Clear, verdict)
}

func TestAShapeWithNoBoundOnlyMeasures(t *testing.T) {
	c := shapeConfig()
	c.Shape.MaxSkew = nil
	c.Shape.CertainSkew = nil
	h := newHarness(c)

	verdict, _ := play(h, 1100, randomSleep)

	assert.Equal(t, detect.Clear, verdict)
}

func TestAPauseDoesNotEndTheShape(t *testing.T) {
	h := newHarness(shapeConfig())

	_, _ = play(h, 450, randomSleep)
	h.clock.Advance(time.Minute)

	verdict, _ := play(h, 100, randomSleep)

	assert.Equal(t, detect.Suspect, verdict, "a minute away is skipped, and the gaps either side still count")
}

func TestTheStrongerRuleIsTheOneReported(t *testing.T) {
	h := newHarness(shapeConfig())

	// A one second loop with a little wobble: cadence reaches Certain and the shape reads only Suspect.
	gaps := []time.Duration{980, 1000, 1020, 990, 1010}
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)
	for range 140 {
		for _, gap := range gaps {
			verdict, evidence = h.after(gap * time.Millisecond)
		}
	}

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "cadence", evidence.Rule)
}
