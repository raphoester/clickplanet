//go:build testing

package activity

import (
	"slices"
	"sync"
)

type Recorded struct {
	mu     sync.Mutex
	events []Event
}

func (r *Recorded) Record(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
}

func (r *Recorded) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.events)
}
