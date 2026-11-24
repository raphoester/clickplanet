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
	Deleted State = "deleted"
)

// Heard is what a module heard of a flag opened since the game shipped: open, closed or deleted, as the latest fact
// told it, and whether planet told the flags module a player played for it.
type Heard struct {
	state     State
	decidedAt time.Time
	kept      bool
}

// Only a later fact changes it: one told again, or one older than the fact it holds, keeps it as it is.
func (h Heard) Opened(at time.Time) Heard {
	return h.decided(Open, at)
}

func (h Heard) Closed(at time.Time) Heard {
	return h.decided(Closed, at)
}

func (h Heard) Deleted(at time.Time) Heard {
	return h.decided(Deleted, at)
}

func (h Heard) decided(state State, at time.Time) Heard {
	if h.state != unheard && !at.After(h.decidedAt) {
		return h
	}
	h.state, h.decidedAt = state, at
	return h
}

// Kept: the flags module was told a player played for it.
func (h Heard) Kept() Heard {
	h.kept = true
	return h
}

func (h Heard) playable() bool {
	return h.state == Open
}

func (h Heard) known() bool {
	return h.state == Open || h.state == Closed
}

func (h Heard) unkept() bool {
	return h.known() && !h.kept
}

func (h Heard) TakeSnapshot() HeardSnapshot {
	return HeardSnapshot{State: h.state, DecidedAt: h.decidedAt, Kept: h.kept}
}

type HeardSnapshot struct {
	State     State
	DecidedAt time.Time
	Kept      bool
}

func (s HeardSnapshot) Restore() (Heard, error) {
	switch s.State {
	case unheard, Open, Closed, Deleted:
		return Heard{state: s.State, decidedAt: s.DecidedAt, kept: s.Kept}, nil
	}
	return Heard{}, fmt.Errorf("a flag heard %q, which is no state", s.State)
}
