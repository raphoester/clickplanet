package evidence

import (
	"context"
	"os"
	"path/filepath"
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
	path   string
	clock  *cptime.FixedClock
	errors []error
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return &harness{
		path:  filepath.Join(t.TempDir(), "evidence.bin"),
		clock: cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)),
	}
}

func (h *harness) store(sections ...Section) *Store {
	return New(Config{StatePath: h.path, Retention: 72 * time.Hour}, h.clock, func(err error) {
		h.errors = append(h.errors, err)
	}, sections...)
}

// shutdown runs the store the way the guard does and stops it, which saves.
func (h *harness) shutdown(store *Store) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store.Run(ctx)
}

func TestEvidenceSurvivesARestart(t *testing.T) {
	h := newHarness(t)

	before := &fake{name: "metronome", events: []time.Time{h.clock.Now().Add(-time.Minute), h.clock.Now()}}
	h.shutdown(h.store(before))

	h.clock.Advance(20 * time.Second)

	after := &fake{name: "metronome"}
	restarted := h.store(after, &fake{name: "jury"})
	restarted.LoadState()

	require.Empty(t, h.errors)
	require.Len(t, after.events, 2)
	assert.True(t, after.events[1].Equal(before.events[1]))

	h.clock.Advance(3 * time.Second)
	restarted.Resume()

	assert.Equal(t, 23*time.Second, after.outage.Length(), "from the save to the moment the new process starts watching")
}

func TestAMissingFileStartsEmptyAndSaysNothing(t *testing.T) {
	h := newHarness(t)

	section := &fake{name: "sequencer"}
	store := h.store(section)
	store.LoadState()
	store.Resume()

	assert.Empty(t, h.errors)
	assert.Empty(t, section.events)
	assert.Zero(t, section.outage, "no file, no outage to bridge")
}

func TestACorruptFileIsReportedAndStartsEmpty(t *testing.T) {
	for name, damage := range map[string]func([]byte) []byte{
		"flipped byte":            func(raw []byte) []byte { raw[len(raw)-1] ^= 0xff; return raw },
		"truncated":               func(raw []byte) []byte { return raw[:len(raw)/2] },
		"bad magic":               func(raw []byte) []byte { raw[0] = 'X'; return raw },
		"shorter than the header": func([]byte) []byte { return []byte("CP") },
		"newer version": func(raw []byte) []byte {
			raw[len(stateMagic)] = stateVersion + 1
			return raw
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.shutdown(h.store(&fake{name: "catcher", events: []time.Time{h.clock.Now()}}))

			raw, err := os.ReadFile(h.path)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(h.path, damage(raw), 0o600))

			section := &fake{name: "catcher"}
			h.store(section).LoadState()

			require.Len(t, h.errors, 1)
			assert.ErrorIs(t, h.errors[0], errCorruptState)
			assert.Empty(t, section.events)
		})
	}
}

func TestASectionThatDoesNotDecodeStartsEmptyAlone(t *testing.T) {
	h := newHarness(t)
	h.shutdown(h.store(
		&fake{name: "retaker", events: []time.Time{h.clock.Now()}},
		&fake{name: "defender", events: []time.Time{h.clock.Now()}},
	))

	wrongShape := &shapeless{name: "retaker"}
	defender := &fake{name: "defender"}
	h.store(wrongShape, defender).LoadState()

	require.Len(t, h.errors, 1)
	assert.Len(t, defender.events, 1)
}

func TestEvidenceOlderThanTheRetentionIsDroppedOnLoad(t *testing.T) {
	h := newHarness(t)

	old := h.clock.Now()
	h.clock.Advance(71 * time.Hour)
	recent := h.clock.Now()
	h.shutdown(h.store(&fake{name: "jury", events: []time.Time{old, recent}}))

	// Down for two hours: the first event is now past three days.
	h.clock.Advance(2 * time.Hour)

	section := &fake{name: "jury"}
	h.store(section).LoadState()

	require.Len(t, section.events, 1)
	assert.True(t, section.events[0].Equal(recent))
}

func TestTheSweepDropsEvidenceOlderThanTheRetentionAndTheFileFollows(t *testing.T) {
	h := newHarness(t)

	section := &fake{name: "jury", events: []time.Time{h.clock.Now()}}
	store := New(Config{StatePath: h.path, Retention: 72 * time.Hour, SaveInterval: 10 * time.Millisecond}, h.clock, nil, section)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		store.Run(ctx)
		close(done)
	}()

	h.clock.Advance(73 * time.Hour)
	assert.Eventually(t, func() bool {
		loaded := &fake{name: "jury", events: []time.Time{h.clock.Now()}}
		h.store(loaded).LoadState()
		return len(loaded.events) == 0
	}, time.Second, 10*time.Millisecond)

	cancel()
	<-done

	assert.Empty(t, section.events)
}

func TestNoPathSavesNothing(t *testing.T) {
	h := newHarness(t)

	store := New(Config{}, h.clock, func(err error) { h.errors = append(h.errors, err) }, &fake{name: "jury"})
	h.shutdown(store)
	store.LoadState()

	assert.Empty(t, h.errors)
	_, err := os.Stat(h.path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

type shapeless struct{ name string }

func (s *shapeless) Name() string { return s.name }

func (s *shapeless) Save() ([]byte, error) { return Encode("") }

func (s *shapeless) Load(data []byte) error {
	var wrong struct{ Nothing map[bool]bool }
	return Decode(data, &wrong)
}

func (s *shapeless) Forget(time.Time) {}
