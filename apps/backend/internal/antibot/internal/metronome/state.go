package metronome

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var (
	_ evidence.Section = (*Watchdog)(nil)
	_ evidence.Resumer = (*Watchdog)(nil)
)

type savedCaller struct {
	Scope     string
	LastSeen  int64
	RunStart  int64
	RunClicks int
	Gaps      []int64
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedCaller, 0, len(w.callers))
	for scope, c := range w.callers {
		gaps := make([]int64, 0, len(c.gaps))
		for _, gap := range c.gaps {
			gaps = append(gaps, int64(gap))
		}
		saved = append(saved, savedCaller{
			Scope:     scope,
			LastSeen:  evidence.Nanos(c.lastSeen),
			RunStart:  evidence.Nanos(c.runStart),
			RunClicks: c.runClicks,
			Gaps:      gaps,
		})
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
		loaded := &caller{
			lastSeen:  evidence.Time(c.LastSeen),
			runStart:  evidence.Time(c.RunStart),
			runClicks: c.RunClicks,
		}
		for _, gap := range c.Gaps {
			loaded.gaps = append(loaded.gaps, time.Duration(gap))
		}
		if capacity := w.capacity(); len(loaded.gaps) > capacity {
			loaded.gaps = loaded.gaps[len(loaded.gaps)-capacity:]
		}
		callers[c.Scope] = loaded
	}

	w.mu.Lock()
	w.callers = callers
	w.mu.Unlock()

	return nil
}

// Forget drops a caller silent since before; a run still going is kept whole, however long ago it started.
func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for scope, c := range w.callers {
		if c.lastSeen.Before(before) {
			delete(w.callers, scope)
		}
	}
}

func (w *Watchdog) Resume(outage detect.Outage) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.outage = outage
}
