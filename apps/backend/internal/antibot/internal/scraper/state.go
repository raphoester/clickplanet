package scraper

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedCaller struct {
	Scope   string
	Slices  []savedSlice
	Clicked int64
}

type savedSlice struct {
	At      int64
	Maps    float64
	Streams int
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedCaller, 0, len(w.callers))
	for scope, c := range w.callers {
		slices := make([]savedSlice, 0, len(c.slices))
		for _, s := range c.slices {
			slices = append(slices, savedSlice{At: evidence.Nanos(s.at), Maps: s.maps, Streams: s.streams})
		}
		saved = append(saved, savedCaller{Scope: scope, Slices: slices, Clicked: evidence.Nanos(c.clicked)})
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
		loaded := &caller{clicked: evidence.Time(c.Clicked)}
		for _, s := range c.Slices {
			loaded.slices = append(loaded.slices, slice{at: evidence.Time(s.At), maps: s.Maps, streams: s.Streams})
		}
		if len(loaded.slices) > 0 {
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
		c.prune(before)

		if len(c.slices) == 0 {
			delete(w.callers, scope)
		}
	}
}
