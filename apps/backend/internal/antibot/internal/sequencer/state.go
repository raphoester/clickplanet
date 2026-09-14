package sequencer

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedCaller struct {
	Scope    string
	LastTile uint32
	Seen     bool
	Steps    []savedStep
}

type savedStep struct {
	At   int64
	Size int64
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedCaller, 0, len(w.callers))
	for scope, c := range w.callers {
		steps := make([]savedStep, 0, len(c.steps))
		for _, s := range c.steps {
			steps = append(steps, savedStep{At: evidence.Nanos(s.at), Size: s.size})
		}
		saved = append(saved, savedCaller{Scope: scope, LastTile: c.lastTile, Seen: c.seen, Steps: steps})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved []savedCaller
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	callers := make(map[string]*caller, len(saved))
	for _, c := range saved {
		loaded := &caller{lastTile: c.LastTile, seen: c.Seen}
		for _, s := range c.Steps {
			loaded.steps = append(loaded.steps, step{at: evidence.Time(s.At), size: s.Size})
		}
		if capacity := w.capacity(); len(loaded.steps) > capacity {
			loaded.steps = loaded.steps[len(loaded.steps)-capacity:]
		}
		callers[c.Scope] = loaded
	}

	w.mu.Lock()
	w.callers = callers
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for scope, c := range w.callers {
		c.prune(before)
		if len(c.steps) == 0 {
			delete(w.callers, scope)
		}
	}
}
