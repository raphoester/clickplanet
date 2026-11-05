//go:build testing

package bonuses

import (
	"context"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	hands   map[Holder]Held
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{hands: map[Holder]Held{}}
}

func (m *MemoryStore) Change(_ context.Context, holder Holder, change func(Held) (Held, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	after, err := change(m.hands[holder])
	if err != nil {
		return err
	}
	if after.Empty() {
		delete(m.hands, holder)
	} else {
		m.hands[holder] = after
	}
	return nil
}

func (m *MemoryStore) Held(_ context.Context, holder Holder) (Held, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.hands[holder], m.failing
}

func (m *MemoryStore) Keep(holder Holder, held Held) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hands[holder] = held
}

func (m *MemoryStore) Kept(holder Holder) Held {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.hands[holder]
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
