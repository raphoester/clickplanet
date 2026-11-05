//go:build testing

package rules

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"google.golang.org/protobuf/proto"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	parts   Parts
	sets    map[SetName]Snapshot
	failing error
}

func NewMemoryStore(parts Parts, kept ...Snapshot) *MemoryStore {
	sets := make(map[SetName]Snapshot, len(kept))
	for _, snapshot := range kept {
		sets[snapshot.Name] = snapshot
	}
	return &MemoryStore{parts: parts, sets: sets}
}

func (m *MemoryStore) Set(_ context.Context, name SetName) (Set, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Set{}, m.failing
	}
	defaults, ok := m.sets[Default]
	if !ok {
		return Set{}, fmt.Errorf("%q: %w", Default, ErrUnknownSet)
	}
	kept, ok := m.sets[name]
	if !ok {
		return Set{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	}
	set, err := kept.Restore(m.parts, defaults)
	if err != nil {
		return Set{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (m *MemoryStore) Names(context.Context) ([]SetName, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	return slices.Sorted(maps.Keys(m.sets)), nil
}

func (m *MemoryStore) Save(_ context.Context, set Set) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	snapshot := set.TakeSnapshot()
	m.sets[snapshot.Name] = snapshot
	return nil
}

// Keep writes a set as it is, one that cannot play too, as a hand edit in SQL would.
func (m *MemoryStore) Keep(kept Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sets[kept.Name] = Snapshot{Name: kept.Name, Rules: proto.CloneOf(kept.Rules)}
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
