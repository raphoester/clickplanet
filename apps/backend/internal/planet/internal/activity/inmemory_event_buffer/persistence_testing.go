//go:build testing

package inmemory_event_buffer

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
)

type MemoryPersistence struct {
	mu     sync.Mutex
	saved  []activity.Event
	saves  int
	broken error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{}
}

var _ Persistence = (*MemoryPersistence)(nil)

func (p *MemoryPersistence) Save(_ context.Context, events []activity.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.broken != nil {
		return p.broken
	}

	p.saves++
	p.saved = append(p.saved, events...)

	return nil
}

func (p *MemoryPersistence) Break(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.broken = err
}

func (p *MemoryPersistence) Heal() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.broken = nil
}

func (p *MemoryPersistence) Saved() []activity.Event {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.saved)
}

func (p *MemoryPersistence) Saves() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.saves
}
