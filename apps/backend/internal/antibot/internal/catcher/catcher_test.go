package catcher_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type harness struct {
	watchdog *catcher.Watchdog
	clock    *cptime.FixedClock
}

func newHarness() *harness {
	h := &harness{clock: cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))}
	h.watchdog = catcher.New(catcher.Config{
		MinCatches:    5,
		MaxMedian:     3 * time.Second,
		CertainMedian: 1500 * time.Millisecond,
		TrackWindow:   30 * time.Minute,
	}, h.clock)
	return h
}

// catches waits the ordinary pace between boxes, then catches one after each delay.
func (h *harness) catches(delays ...time.Duration) {
	for _, after := range delays {
		h.clock.Advance(2 * time.Minute)
		h.watchdog.Caught("caller", after)
	}
}

func (h *harness) miss() {
	h.clock.Advance(2 * time.Minute)
	h.watchdog.Missed("caller")
}

func (h *harness) click() (detect.Verdict, detect.Evidence) {
	h.clock.Advance(time.Second)
	return h.watchdog.Watch(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: h.clock.Now()})
}

func TestEveryBoxCaughtAtOnceIsCertain(t *testing.T) {
	h := newHarness()

	h.catches(300*time.Millisecond, 250*time.Millisecond, 400*time.Millisecond, 280*time.Millisecond, 320*time.Millisecond)

	verdict, evidence := h.click()

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "catch", evidence.Rule)
}

func TestEveryBoxCaughtWithinThreeSecondsIsOnlySuspect(t *testing.T) {
	h := newHarness()

	h.catches(2*time.Second, 2500*time.Millisecond, 1800*time.Millisecond, 2200*time.Millisecond, 2900*time.Millisecond)

	verdict, _ := h.click()

	assert.Equal(t, detect.Suspect, verdict)
}

func TestAPlayerWhoTakesTimeToFindTheBoxIsClear(t *testing.T) {
	h := newHarness()

	h.catches(6*time.Second, 4*time.Second, 9*time.Second, 1*time.Second, 5*time.Second)

	verdict, _ := h.click()

	assert.Equal(t, detect.Clear, verdict, "one lucky fast catch is not the median")
}

func TestOneMissClearsAFastCatcher(t *testing.T) {
	h := newHarness()

	h.catches(300*time.Millisecond, 300*time.Millisecond)
	h.miss()
	h.catches(300*time.Millisecond, 300*time.Millisecond)

	verdict, _ := h.click()
	assert.Equal(t, detect.Clear, verdict, "a caller who misses boxes is not catching all of them")

	h.catches(300 * time.Millisecond)

	verdict, _ = h.click()
	assert.Equal(t, detect.Clear, verdict, "the miss is still one of the last five")

	h.catches(300*time.Millisecond, 300*time.Millisecond)

	verdict, _ = h.click()
	assert.Equal(t, detect.Certain, verdict, "five in a row since the miss")
}

func TestTooFewBoxesSayNothing(t *testing.T) {
	h := newHarness()

	h.catches(300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond)

	verdict, _ := h.click()

	assert.Equal(t, detect.Clear, verdict)
}

func TestCatchesOlderThanTheWindowStopCounting(t *testing.T) {
	h := newHarness()

	h.catches(300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond)
	h.clock.Advance(time.Hour)

	verdict, _ := h.click()

	assert.Equal(t, detect.Clear, verdict)
}

func TestAnotherCallersCatchesAreNotTheirs(t *testing.T) {
	h := newHarness()

	h.catches(300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond)

	verdict, _ := h.watchdog.Watch(detect.Click{Scope: "someone else", Tile: 1, Country: "FR", At: h.clock.Now()})

	assert.Equal(t, detect.Clear, verdict)
}
