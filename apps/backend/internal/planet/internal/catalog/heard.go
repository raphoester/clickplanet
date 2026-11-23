package catalog

import (
	"fmt"
	"time"
)

type State string

const (
	unheard State = ""
	Open    State = "open"
	Closed  State = "closed"
)

// Heard is what a module heard of a flag opened since the game shipped: open or closed, as the latest fact told it.
type Heard struct {
	state     State
	decidedAt time.Time
}

// Only a later fact changes it: one told again, or one older than the fact it holds, keeps it as it is.
func (h Heard) Opened(at time.Time) Heard {
	return h.decided(Open, at)
}

func (h Heard) Closed(at time.Time) Heard {
	return h.decided(Closed, at)
}

func (h Heard) decided(state State, at time.Time) Heard {
	if h.state != unheard && !at.After(h.decidedAt) {
		return h
	}
	return Heard{state: state, decidedAt: at}
}

func (h Heard) playable() bool {
	return h.state == Open
}

func (h Heard) known() bool {
	return h.state != unheard
}

func (h Heard) TakeSnapshot() HeardSnapshot {
	return HeardSnapshot{State: h.state, DecidedAt: h.decidedAt}
}

type HeardSnapshot struct {
	State     State
	DecidedAt time.Time
}

func (s HeardSnapshot) Restore() (Heard, error) {
	switch s.State {
	case unheard, Open, Closed:
		return Heard{state: s.State, decidedAt: s.DecidedAt}, nil
	}
	return Heard{}, fmt.Errorf("a flag heard %q, which is no state", s.State)
}
