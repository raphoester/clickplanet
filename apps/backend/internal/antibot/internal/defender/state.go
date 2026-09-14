package defender

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedState struct {
	Losses  []savedLoss
	Callers []savedCaller
}

type savedLoss struct {
	Tile    uint32
	Country string
	To      string
	At      int64
}

type savedCaller struct {
	Scope string
	Takes []savedTake
}

type savedTake struct {
	At     int64
	Retake bool
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := savedState{
		Losses:  make([]savedLoss, 0, len(w.losses)),
		Callers: make([]savedCaller, 0, len(w.callers)),
	}
	for tile, l := range w.losses {
		saved.Losses = append(saved.Losses, savedLoss{Tile: tile, Country: l.country, To: l.to, At: evidence.Nanos(l.at)})
	}
	for scope, c := range w.callers {
		takes := make([]savedTake, 0, len(c.takes))
		for _, t := range c.takes {
			takes = append(takes, savedTake{At: evidence.Nanos(t.at), Retake: t.retake})
		}
		saved.Callers = append(saved.Callers, savedCaller{Scope: scope, Takes: takes})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved savedState
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	losses := make(map[uint32]loss, len(saved.Losses))
	for _, l := range saved.Losses {
		losses[l.Tile] = loss{country: l.Country, to: l.To, at: evidence.Time(l.At)}
	}

	callers := make(map[string]*caller, len(saved.Callers))
	for _, c := range saved.Callers {
		loaded := &caller{}
		for _, t := range c.Takes {
			loaded.takes = append(loaded.takes, take{at: evidence.Time(t.At), retake: t.Retake})
		}
		if len(loaded.takes) > maxSamples {
			loaded.takes = loaded.takes[len(loaded.takes)-maxSamples:]
		}
		callers[c.Scope] = loaded
	}

	w.mu.Lock()
	w.losses, w.callers = losses, callers
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for tile, l := range w.losses {
		if l.at.Before(before) {
			delete(w.losses, tile)
		}
	}

	for scope, c := range w.callers {
		c.prune(before)
		if len(c.takes) == 0 {
			delete(w.callers, scope)
		}
	}
}
