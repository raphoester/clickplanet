//go:build testing

package rules

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	sets    map[SetName]Snapshot
	failing error
}

func NewMemoryStore(kept ...Snapshot) *MemoryStore {
	sets := make(map[SetName]Snapshot, len(kept))
	for _, snapshot := range kept {
		sets[snapshot.Name] = snapshot
	}
	return &MemoryStore{sets: sets}
}

func (m *MemoryStore) Set(_ context.Context, name SetName) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Snapshot{}, m.failing
	}
	kept, ok := m.sets[name]
	if !ok {
		return Snapshot{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	}
	return Snapshot{Name: kept.Name, Rules: proto.CloneOf(kept.Rules)}, nil
}

func (m *MemoryStore) Save(_ context.Context, set Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.sets[set.Name] = Snapshot{Name: set.Name, Rules: proto.CloneOf(set.Rules)}
	return nil
}

func (m *MemoryStore) Kept(name SetName) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept, ok := m.sets[name]
	return kept, ok
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
