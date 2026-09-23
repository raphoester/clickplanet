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

func newForeignHarness() *harness {
	h := &harness{clock: cptime.NewFixedClock(time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))}
	h.watchdog = catcher.New(catcher.Config{
		MinCatches:    5,
		MaxMedian:     3 * time.Second,
		CertainMedian: 1500 * time.Millisecond,
		TrackWindow:   time.Hour,
		Foreign:       catcher.ForeignConfig{Window: time.Hour, MinClaims: 5, CertainClaims: 10},
	}, h.clock)
	return h
}

// foreign claims n boxes that were never the caller's, every gap.
func (h *harness) foreign(n int, gap time.Duration) {
	for range n {
		h.clock.Advance(gap)
		h.watchdog.Foreign("caller")
	}
}

func TestClaimingOtherCallersBoxesIsSuspectThenCertain(t *testing.T) {
	h := newForeignHarness()

	h.foreign(4, time.Minute)
	verdict, _ := h.click()
	require.Equal(t, detect.Clear, verdict)

	h.foreign(1, time.Minute)
	verdict, evidence := h.click()
	require.Equal(t, detect.Suspect, verdict)
	assert.Equal(t, "foreign claims=5 within=1h0m0s", evidence.String())

	h.foreign(5, time.Minute)
	verdict, _ = h.click()
	assert.Equal(t, detect.Certain, verdict)
}

func TestForeignClaimsOlderThanTheWindowStopCounting(t *testing.T) {
	h := newForeignHarness()

	h.foreign(10, time.Minute)
	h.clock.Advance(time.Hour)

	verdict, _ := h.click()

	assert.Equal(t, detect.Clear, verdict)
}

func TestTheStrongerRuleIsReported(t *testing.T) {
	h := newForeignHarness()

	h.catches(2*time.Second, 2*time.Second, 2*time.Second, 2*time.Second, 2*time.Second)
	h.foreign(10, time.Second)

	verdict, evidence := h.click()

	assert.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "foreign", evidence.Rule)
}

func TestForeignClaimsAreIgnoredWhileNoBoundIsSet(t *testing.T) {
	h := newHarness()

	h.foreign(50, time.Second)
	verdict, _ := h.click()

	assert.Equal(t, detect.Clear, verdict)
}

func TestForeignClaimsSurviveARestart(t *testing.T) {
	h := newForeignHarness()
	h.foreign(10, time.Minute)

	saved, err := h.watchdog.Save()
	require.NoError(t, err)

	restarted := newForeignHarness()
	restarted.clock = h.clock
	require.NoError(t, restarted.watchdog.Load(saved))

	verdict, _ := restarted.click()
	assert.Equal(t, detect.Certain, verdict)
}
