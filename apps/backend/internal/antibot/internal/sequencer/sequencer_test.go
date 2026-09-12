package sequencer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type harness struct {
	watchdog *sequencer.Watchdog
	clock    *fakeClock
}

func newHarness(config sequencer.Config) *harness {
	h := &harness{clock: &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}}
	h.watchdog = sequencer.New(config, h.clock)
	return h
}

func (h *harness) click(tile uint32) (detect.Verdict, detect.Evidence) {
	verdict, evidence := h.watchdog.Watch(detect.Click{
		Scope:   "caller",
		Tile:    tile,
		Country: "FR",
		At:      h.clock.now,
	})

	h.clock.now = h.clock.now.Add(time.Second)

	return verdict, evidence
}

// walk clicks count tiles, each stride on from the last.
func (h *harness) walk(first uint32, stride uint32, count int) (detect.Verdict, detect.Evidence) {
	var (
		verdict  detect.Verdict
		evidence detect.Evidence
	)

	for i := range count {
		verdict, evidence = h.click(first + uint32(i)*stride)
	}

	return verdict, evidence
}

func config() sequencer.Config {
	return sequencer.Config{
		MinSteps:     10,
		MinShare:     0.75,
		CertainSteps: 40,
		CertainShare: 0.95,
		TrackWindow:  time.Hour,
	}
}

func TestWalkingTheIdsForLongEnoughIsCertain(t *testing.T) {
	h := newHarness(config())

	verdict, evidence := h.walk(1000, 1, 60)

	require.Equal(t, detect.Certain, verdict)
	assert.Equal(t, "stride", evidence.Rule)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "stride", Value: int64(1)})
}

func TestAShortWalkIsOnlySuspect(t *testing.T) {
	h := newHarness(config())

	verdict, _ := h.walk(2000, 1, 15)

	assert.Equal(t, detect.Suspect, verdict, "fifteen tidy clicks is not yet a machine")
}

func TestAnyConstantStrideIsAWalk(t *testing.T) {
	h := newHarness(config())

	// The size of the step says nothing. Holding one says everything.
	verdict, evidence := h.walk(3000, 7, 60)

	require.Equal(t, detect.Certain, verdict)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "stride", Value: int64(7)})
}

func TestWalkingBackwardsCountsToo(t *testing.T) {
	h := newHarness(config())

	for i := range 60 {
		h.click(uint32(9000 - i))
	}

	verdict, evidence := h.click(8940)

	require.Equal(t, detect.Certain, verdict)
	assert.Contains(t, evidence.Fields, detect.Field{Key: "stride", Value: int64(-1)})
}

func TestOneBreakInTheRunDoesNotSaveIt(t *testing.T) {
	h := newHarness(config())

	// A sweep that reaches the end of a band and jumps to the next is still a
	// sweep: one odd step out of sixty changes no share worth speaking of.
	h.walk(4000, 1, 30)
	h.click(50000)
	verdict, _ := h.walk(50001, 1, 30)

	assert.Equal(t, detect.Certain, verdict)
}

func TestAHandWanderingIsClear(t *testing.T) {
	h := newHarness(config())

	// Tile ids follow the icosahedron's vertex order, so filling in a shape by
	// hand does not hold a step between one click and the next.
	steps := []int{3, -1, 12, 2, -7, 1, 40, -3, 5, 1, -22, 8, 2, 17, -4, 1, 9, -13, 6, 2}

	tile := uint32(5000)
	var verdict detect.Verdict
	for range 4 {
		for _, step := range steps {
			tile = uint32(int(tile) + step)
			verdict, _ = h.click(tile)
		}
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestLeaningOnOneTileIsNotAWalk(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for range 60 {
		verdict, _ = h.click(6000)
	}

	assert.Equal(t, detect.Clear, verdict, "a caller sitting on one tile is the throttle's problem")
}

func TestFightingOverTwoTilesIsNotAWalk(t *testing.T) {
	h := newHarness(config())

	var verdict detect.Verdict
	for i := range 60 {
		verdict, _ = h.click(uint32(7000 + i%2))
	}

	assert.Equal(t, detect.Clear, verdict)
}

func TestStepsAgeOutOfTheWindow(t *testing.T) {
	c := config()
	c.TrackWindow = time.Minute

	h := newHarness(c)

	require.Equal(t, detect.Certain, func() detect.Verdict { v, _ := h.walk(8000, 1, 60); return v }())

	h.clock.now = h.clock.now.Add(10 * time.Minute)

	verdict, _ := h.click(8100)
	assert.Equal(t, detect.Clear, verdict, "a walk from an hour ago is not a walk now")
}
