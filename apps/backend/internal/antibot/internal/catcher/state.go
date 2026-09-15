package catcher

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedCaller struct {
	Scope    string
	Outcomes []savedOutcome
}

type savedOutcome struct {
	At     int64
	Caught bool
	After  int64
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedCaller, 0, len(w.callers))
	for scope, c := range w.callers {
		outcomes := make([]savedOutcome, 0, len(c.outcomes))
		for _, o := range c.outcomes {
			outcomes = append(outcomes, savedOutcome{At: evidence.Nanos(o.at), Caught: o.caught, After: int64(o.after)})
		}
		saved = append(saved, savedCaller{Scope: scope, Outcomes: outcomes})
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
		loaded := &caller{}
		for _, o := range c.Outcomes {
			loaded.outcomes = append(loaded.outcomes, outcome{at: evidence.Time(o.At), caught: o.Caught, after: time.Duration(o.After)})
		}
		if len(loaded.outcomes) > w.config.MinCatches {
			loaded.outcomes = loaded.outcomes[len(loaded.outcomes)-w.config.MinCatches:]
		}
		if len(loaded.outcomes) > 0 {
			callers[c.Scope] = loaded
		}
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
		kept := c.outcomes[:0]
		for _, o := range c.outcomes {
			if !o.at.Before(before) {
				kept = append(kept, o)
			}
		}
		c.outcomes = kept

		if len(c.outcomes) == 0 {
			delete(w.callers, scope)
		}
	}
}
