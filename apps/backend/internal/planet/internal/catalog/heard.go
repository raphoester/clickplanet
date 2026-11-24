package catalog

import (
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type State string

const (
	unheard State = ""
	Open    State = "open"
	Closed  State = "closed"
)

// Heard is what a module heard of a flag opened since the game shipped: open or closed, as the latest fact told it, and
// its pin, as the latest fact that named one told it.
type Heard struct {
	state     State
	decidedAt time.Time
	pin       clicks.Pin
	pinnedAt  time.Time
}

// Only a later fact changes it: one told again, or one older than the fact it holds, keeps it as it is.
func (h Heard) Opened(at time.Time) Heard {
	return h.decided(Open, at)
}

func (h Heard) Closed(at time.Time) Heard {
	return h.decided(Closed, at)
}

// Moved keeps no pin of a flag never heard. A pin told at the time of the one it holds replaces it: the restatement at
// each boot puts a pin right.
func (h Heard) Moved(pin clicks.Pin, at time.Time) Heard {
	if h.state == unheard || (!h.pinnedAt.IsZero() && at.Before(h.pinnedAt)) {
		return h
	}
	h.pin, h.pinnedAt = pin, at
	return h
}

func (h Heard) decided(state State, at time.Time) Heard {
	if h.state != unheard && !at.After(h.decidedAt) {
		return h
	}
	h.state, h.decidedAt = state, at
	return h
}

func (h Heard) playable() bool {
	return h.state == Open
}

func (h Heard) known() bool {
	return h.state != unheard
}

// seated is the pin of a flag open now.
func (h Heard) seated() (clicks.Pin, bool) {
	return h.pin, h.playable() && !h.pinnedAt.IsZero()
}

func (h Heard) TakeSnapshot() HeardSnapshot {
	return HeardSnapshot{State: h.state, DecidedAt: h.decidedAt, Pin: h.pin, PinnedAt: h.pinnedAt}
}

// PinnedAt is zero for a flag no fact gave a pin yet.
type HeardSnapshot struct {
	State     State
	DecidedAt time.Time
	Pin       clicks.Pin
	PinnedAt  time.Time
}

func (s HeardSnapshot) Restore() (Heard, error) {
	switch s.State {
	case unheard, Open, Closed:
		return Heard{state: s.State, decidedAt: s.DecidedAt, pin: s.Pin, pinnedAt: s.PinnedAt}, nil
	}
	return Heard{}, fmt.Errorf("a flag heard %q, which is no state", s.State)
}
