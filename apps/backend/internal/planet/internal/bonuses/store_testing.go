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
	before := m.hands[holder]
	after, err := change(before)
	if err != nil {
		return err
	}
	if after == before {
		return nil
	}
	if after.Empty() {
		delete(m.hands, holder)
		return nil
	}
	m.hands[holder] = after
	return nil
}

func (m *MemoryStore) Held(_ context.Context, holder Holder) (Held, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Held{}, m.failing
	}
	return m.hands[holder], nil
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
