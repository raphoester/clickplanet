package hopper_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type harness struct {
	watchdog *hopper.Watchdog
	clock    *cptime.FixedClock
	rng      *rand.Rand
}

func newHarness(config hopper.Config) *harness {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC))
	return &harness{
		watchdog: hopper.New(config, clock, func(float64) {}),
		clock:    clock,
		//nolint:gosec // seeded test PRNG
		rng: rand.New(rand.NewPCG(10, 9)),
	}
}

func config() hopper.Config {
	return hopper.Config{
		MinAngle:     45,
		MinGap:       100 * time.Millisecond,
		MaxGap:       30 * time.Second,
		MinSteps:     40,
		MinShare:     0.4,
		CertainSteps: 200,
		CertainShare: 0.7,
		TrackWindow:  15 * time.Minute,
	}
}

func (h *harness) clickAs(scope, account string, at detect.Point) detect.Verdict {
	click := detect.Click{Scope: scope, Account: account, Position: at, At: h.clock.Now()}
	h.watchdog.Attempted(click)
	verdict, _ := h.watchdog.Watch(click)
	return verdict
}

func (h *harness) click(at detect.Point) detect.Verdict {
	return h.clickAs("2001:db8::/64", "player", at)
}

func (h *harness) anywhere() detect.Point {
	z := 2*h.rng.Float64() - 1
	phi := 2 * math.Pi * h.rng.Float64()
	r := math.Sqrt(1 - z*z)
	return detect.Point{X: r * math.Cos(phi), Y: r * math.Sin(phi), Z: z}
}

func (h *harness) near(centre detect.Point) detect.Point {
	jitter := func() float64 { return 0.02 * (h.rng.Float64() - 0.5) }
	return detect.Point{X: centre.X + jitter(), Y: centre.Y + jitter(), Z: centre.Z + jitter()}
}

var (
	poland     = detect.Point{X: 0.37, Y: 0.79, Z: 0.12}
	antarctica = detect.Point{X: 0.1, Y: -0.98, Z: 0.1}
	brazil     = detect.Point{X: 0.1, Y: -0.17, Z: -0.98}
)

func TestAClickerAimingAllOverTheGlobeIsCaught(t *testing.T) {
	h := newHarness(config())

	for range 40 {
		h.clock.Advance(1500 * time.Millisecond)
		require.Equal(t, detect.Clear, h.click(h.anywhere()), "too few steps to say anything")
	}

	h.clock.Advance(1500 * time.Millisecond)
	assert.Equal(t, detect.Suspect, h.click(h.anywhere()))

	var verdict detect.Verdict
	for range 160 {
		h.clock.Advance(1500 * time.Millisecond)
		verdict = h.click(h.anywhere())
	}

	assert.Equal(t, detect.Certain, verdict)
}

func TestABotWaitingForEachRefillIsStillRead(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 41 {
		h.clock.Advance(5 * time.Second)
		verdict = h.click(h.anywhere())
	}

	assert.Equal(t, detect.Suspect, verdict, "five seconds is no time to turn the globe for every click")
}

func TestPaintingOneCountryIsClear(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 300 {
		h.clock.Advance(700 * time.Millisecond)
		verdict = h.click(h.near(poland))
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestAPlayerMovingBetweenFrontsIsClear(t *testing.T) {
	h := newHarness(config())
	fronts := []detect.Point{poland, antarctica, brazil}

	var verdict detect.Verdict
	for i := range 400 {
		h.clock.Advance(800 * time.Millisecond)
		verdict = h.click(h.near(fronts[(i/8)%len(fronts)]))
		require.Equal(t, detect.Clear, verdict, "one jump in eight is a player changing fronts, zoomed out")
	}
}

func TestAFlushOfHeldTapsCountsOnlyItsEnds(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 10 {
		for range 20 {
			h.clock.Advance(time.Second)
			verdict = h.click(h.near(poland))
		}

		h.clock.Advance(time.Second)
		for range 20 {
			h.clock.Advance(5 * time.Millisecond)
			verdict = h.click(h.anywhere())
		}
	}

	assert.Equal(t, detect.Clear, verdict, "taps held through a dead network arrive together, whenever they were made")
}

func TestClicksFurtherApartThanMaxGapAreNoStep(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 100 {
		h.clock.Advance(40 * time.Second)
		verdict = h.click(h.anywhere())
	}

	assert.Equal(t, detect.Clear, verdict, "given time, a hand turns the globe anywhere")
}

func TestPlayersBehindOneAddressAreJudgedApart(t *testing.T) {
	t.Run("two accounts", func(t *testing.T) {
		h := newHarness(config())

		for range 100 {
			h.clock.Advance(500 * time.Millisecond)
			require.Equal(t, detect.Clear, h.clickAs("192.0.2.1", "cadet", h.near(poland)))

			h.clock.Advance(500 * time.Millisecond)
			require.Equal(t, detect.Clear, h.clickAs("192.0.2.1", "classmate", h.near(brazil)))
		}
	})

	t.Run("no account to tell them apart", func(t *testing.T) {
		h := newHarness(config())

		var verdict detect.Verdict
		for range 100 {
			h.clock.Advance(500 * time.Millisecond)
			h.clickAs("192.0.2.1", "", h.near(poland))

			h.clock.Advance(500 * time.Millisecond)
			verdict = h.clickAs("192.0.2.1", "", h.near(brazil))
		}

		assert.Equal(t, detect.Suspect, verdict, "one payer, so one hand")
	})
}

func TestATryWithNoPositionIsNoStep(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 100 {
		h.clock.Advance(time.Second)
		verdict = h.click(detect.Point{})
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestTheEvidenceNamesTheRule(t *testing.T) {
	h := newHarness(config())

	for range 41 {
		h.clock.Advance(time.Second)
		h.click(h.anywhere())
	}

	verdict, evidence := h.watchdog.Watch(detect.Click{Scope: "2001:db8::/64", Account: "player", At: h.clock.Now()})

	require.Equal(t, detect.Suspect, verdict)
	assert.Equal(t, "hop", evidence.Rule)
	assert.Contains(t, evidence.String(), "steps=40")
}
