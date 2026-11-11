//go:build testing

package cprules

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"google.golang.org/protobuf/proto"
)

type MemoryStore[M proto.Message] struct {
	parts Parts[M]

	mu      sync.Mutex
	sets    map[SetName]Snapshot[M]
	failing error
}

func NewMemoryStore[M proto.Message](parts Parts[M], kept ...Snapshot[M]) *MemoryStore[M] {
	sets := make(map[SetName]Snapshot[M], len(kept))
	for _, snapshot := range kept {
		sets[snapshot.Name] = snapshot
	}
	return &MemoryStore[M]{parts: parts, sets: sets}
}

func (m *MemoryStore[M]) Set(_ context.Context, name SetName) (Set[M], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Set[M]{}, m.failing
	}
	defaults, ok := m.sets[Default]
	if !ok {
		return Set[M]{}, fmt.Errorf("%q: %w", Default, ErrUnknownSet)
	}
	kept, ok := m.sets[name]
	if !ok {
		return Set[M]{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	}

	set, err := kept.Restore(m.parts, defaults)
	if err != nil {
		return Set[M]{}, fmt.Errorf("the rule set %q kept cannot play: %w", name, err)
	}
	return set, nil
}

func (m *MemoryStore[M]) Names(context.Context) ([]SetName, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	return slices.Sorted(maps.Keys(m.sets)), nil
}

func (m *MemoryStore[M]) Save(_ context.Context, set Set[M]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.sets[set.name] = set.TakeSnapshot()
	return nil
}

// Keep puts a snapshot in as it is, even one that cannot play, as a hand edit in SQL would.
func (m *MemoryStore[M]) Keep(kept Snapshot[M]) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sets[kept.Name] = Snapshot[M]{Name: kept.Name, Rules: proto.CloneOf(kept.Rules)}
}

func (m *MemoryStore[M]) Kept(name SetName) (Snapshot[M], bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept, ok := m.sets[name]
	return kept, ok
}

func (m *MemoryStore[M]) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
