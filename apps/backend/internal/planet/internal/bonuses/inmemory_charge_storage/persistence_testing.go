//go:build testing

package inmemory_charge_storage

import (
	"context"
	"maps"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// MemoryPersistence is a Persistence held in a map, for tests that need charges but not postgres.
type MemoryPersistence struct {
	mu      sync.Mutex
	hands   map[bonuses.Holder]bonuses.Held
	saves   int
	failing error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{hands: map[bonuses.Holder]bonuses.Held{}}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(bonuses.Holder, bonuses.Held)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for holder, hand := range m.hands {
		visit(holder, hand)
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, hands map[bonuses.Holder]bonuses.Held) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves++

	for holder, hand := range hands {
		if hand == (bonuses.Held{}) {
			delete(m.hands, holder)
			continue
		}
		m.hands[holder] = hand
	}
	return nil
}

// Stored is every hand held.
func (m *MemoryPersistence) Stored() map[bonuses.Holder]bonuses.Held {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.hands)
}

func (m *MemoryPersistence) Saves() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saves
}

// FailWith makes every Load and Save return err until Heal.
func (m *MemoryPersistence) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

func (m *MemoryPersistence) Heal() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = nil
}
