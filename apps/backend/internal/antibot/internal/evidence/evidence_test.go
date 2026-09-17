package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fake struct {
	name   string
	events []time.Time
	outage detect.Outage
}

func (f *fake) Name() string { return f.name }

func (f *fake) Save() ([]byte, error) {
	nanos := make([]int64, 0, len(f.events))
	for _, at := range f.events {
		nanos = append(nanos, Nanos(at))
	}
	return Encode(nanos)
}

func (f *fake) Load(data []byte) error {
	var nanos []int64
	if err := Decode(data, &nanos); err != nil {
		return err
	}
	f.events = f.events[:0]
	for _, n := range nanos {
		f.events = append(f.events, Time(n))
	}
	return nil
}

func (f *fake) Forget(before time.Time) {
	kept := f.events[:0]
	for _, at := range f.events {
		if !at.Before(before) {
			kept = append(kept, at)
		}
	}
	f.events = kept
}

func (f *fake) Resume(outage detect.Outage) { f.outage = outage }

type harness struct {
	persistence *MemoryPersistence
	clock       *cptime.FixedClock
	errors      []error
}

func newHarness() *harness {
	return &harness{
		persistence: NewMemoryPersistence(),
		clock:       cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)),
	}
}

func (h *harness) store(sections ...Section) *Store {
	return h.storeWith(Config{Retention: 72 * time.Hour}, sections...)
}

func (h *harness) storeWith(config Config, sections ...Section) *Store {
	return New(config, h.clock, h.persistence, func(err error) {
		h.errors = append(h.errors, err)
	}, sections...)
}

// shutdown runs the store the way the guard does and stops it, which flushes.
func (h *harness) shutdown(store *Store) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store.Run(ctx)
}

func TestEvidenceSurvivesARestart(t *testing.T) {
	h := newHarness()

	before := &fake{name: "metronome", events: []time.Time{h.clock.Now().Add(-time.Minute), h.clock.Now()}}
	h.shutdown(h.store(before))

	h.clock.Advance(20 * time.Second)

	after := &fake{name: "metronome"}
	restarted := h.store(after, &fake{name: "jury"})
	require.NoError(t, restarted.Load(t.Context()))

	require.Empty(t, h.errors)
	require.Len(t, after.events, 2)
	assert.True(t, after.events[1].Equal(before.events[1]))

	h.clock.Advance(3 * time.Second)
	restarted.Resume()

	assert.Equal(t, 23*time.Second, after.outage.Length(), "from the save to the moment the new process starts watching")
}

func TestNothingStoredStartsEmptyAndSaysNothing(t *testing.T) {
	h := newHarness()

	section := &fake{name: "sequencer"}
	store := h.store(section)
	require.NoError(t, store.Load(t.Context()))
	store.Resume()

	assert.Empty(t, h.errors)
	assert.Empty(t, section.events)
	assert.Zero(t, section.outage, "nothing stored, no outage to bridge")
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	h := newHarness()
	h.persistence.FailWith(errors.New("postgres is down"))

	require.Error(t, h.store(&fake{name: "jury"}).Load(t.Context()))
}

func TestAFailedFlushIsReportedAndTheNextOneWritesEverything(t *testing.T) {
	h := newHarness()
	section := &fake{name: "jury", events: []time.Time{h.clock.Now()}}
	store := h.store(section)

	h.persistence.FailWith(errors.New("postgres is down"))
	require.Error(t, store.Flush(t.Context()))

	h.persistence.Heal()
	require.NoError(t, store.Flush(t.Context()))

	loaded := &fake{name: "jury"}
	require.NoError(t, h.store(loaded).Load(t.Context()))
	assert.Len(t, loaded.events, 1)
}

func TestASectionThatDoesNotDecodeStartsEmptyAlone(t *testing.T) {
	h := newHarness()
	h.shutdown(h.store(
		&fake{name: "retaker", events: []time.Time{h.clock.Now()}},
		&fake{name: "defender", events: []time.Time{h.clock.Now()}},
	))

	wrongShape := &shapeless{name: "retaker"}
	defender := &fake{name: "defender"}
	require.NoError(t, h.store(wrongShape, defender).Load(t.Context()))

	require.Len(t, h.errors, 1)
	assert.Len(t, defender.events, 1)
}

func TestEvidenceOlderThanTheRetentionIsDroppedOnLoad(t *testing.T) {
	h := newHarness()

	old := h.clock.Now()
	h.clock.Advance(71 * time.Hour)
	recent := h.clock.Now()
	h.shutdown(h.store(&fake{name: "jury", events: []time.Time{old, recent}}))

	// Down for two hours: the first event is now past three days.
	h.clock.Advance(2 * time.Hour)

	section := &fake{name: "jury"}
	require.NoError(t, h.store(section).Load(t.Context()))

	require.Len(t, section.events, 1)
	assert.True(t, section.events[0].Equal(recent))
}

func TestTheSweepDropsEvidenceOlderThanTheRetentionAndTheTableFollows(t *testing.T) {
	h := newHarness()

	section := &fake{name: "jury", events: []time.Time{h.clock.Now()}}
	store := h.storeWith(Config{Retention: 72 * time.Hour, SaveInterval: 10 * time.Millisecond}, section)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		store.Run(ctx)
		close(done)
	}()

	h.clock.Advance(73 * time.Hour)
	saves := h.persistence.Saves()
	assert.Eventually(t, func() bool { return h.persistence.Saves() > saves+1 }, time.Second, 10*time.Millisecond)

	cancel()
	<-done

	assert.Empty(t, section.events)
	loaded := &fake{name: "jury", events: []time.Time{h.clock.Now()}}
	require.NoError(t, h.store(loaded).Load(t.Context()))
	assert.Empty(t, loaded.events)
}

type shapeless struct{ name string }

func (s *shapeless) Name() string { return s.name }

func (s *shapeless) Save() ([]byte, error) { return Encode("") }

func (s *shapeless) Load(data []byte) error {
	var wrong struct{ Nothing map[bool]bool }
	return Decode(data, &wrong)
}

func (s *shapeless) Forget(time.Time) {}
