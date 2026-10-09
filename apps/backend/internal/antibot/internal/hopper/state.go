package hopper

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedState struct {
	Payers []savedPayer
}

type savedPayer struct {
	Payer string
	Steps []savedStep
}

type savedStep struct {
	At  int64
	Hop bool
}

// Save keeps the steps and not the last click: a step stitched across a restart is not a sample.
func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := savedState{Payers: make([]savedPayer, 0, len(w.payers))}
	for key, p := range w.payers {
		if len(p.steps) == 0 {
			continue
		}
		steps := make([]savedStep, 0, len(p.steps))
		for _, s := range p.steps {
			steps = append(steps, savedStep{At: evidence.Nanos(s.at), Hop: s.hop})
		}
		saved.Payers = append(saved.Payers, savedPayer{Payer: key, Steps: steps})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved savedState
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	capacity := w.capacity()
	payers := make(map[string]*payer, len(saved.Payers))
	for _, p := range saved.Payers {
		loaded := &payer{}
		for _, s := range p.Steps {
			loaded.steps = append(loaded.steps, step{at: evidence.Time(s.At), hop: s.Hop})
		}
		if len(loaded.steps) > capacity {
			loaded.steps = loaded.steps[len(loaded.steps)-capacity:]
		}
		payers[p.Payer] = loaded
	}

	w.mu.Lock()
	w.payers = payers
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for key, p := range w.payers {
		p.prune(before)
		if len(p.steps) == 0 && p.lastAt.Before(before) {
			delete(w.payers, key)
		}
	}
}
