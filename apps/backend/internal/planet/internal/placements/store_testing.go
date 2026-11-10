//go:build testing

package placements

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	kept    map[clicks.PlacementID]Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{kept: map[clicks.PlacementID]Snapshot{}}
}

func (m *MemoryStore) Find(_ context.Context, ids ...clicks.PlacementID) ([]Placement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var found []Placement
	for _, snapshot := range m.oldestFirstLocked() {
		if !slices.Contains(ids, snapshot.ID) {
			continue
		}
		placement, err := snapshot.Restore()
		if err != nil {
			return nil, err
		}
		found = append(found, placement)
	}
	return found, nil
}

func (m *MemoryStore) All(context.Context) ([]clicks.PlacementID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var ids []clicks.PlacementID
	for _, snapshot := range m.oldestFirstLocked() {
		ids = append(ids, snapshot.ID)
	}
	return ids, nil
}

func (m *MemoryStore) Save(ctx context.Context, placements ...Placement) error {
	if err := m.failure(); err != nil {
		return err
	}
	snapshots := make([]Snapshot, 0, len(placements))
	for _, placement := range placements {
		snapshots = append(snapshots, placement.TakeSnapshot())
	}
	cpatomic.OnCommit(ctx, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		for _, snapshot := range snapshots {
			if snapshot.Ended {
				delete(m.kept, snapshot.ID)
				continue
			}
			m.kept[snapshot.ID] = snapshot
		}
	})
	return nil
}

func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.oldestFirstLocked()
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

func (m *MemoryStore) failure() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.failing
}

func (m *MemoryStore) oldestFirstLocked() []Snapshot {
	snapshots := make([]Snapshot, 0, len(m.kept))
	for _, snapshot := range m.kept {
		snapshots = append(snapshots, snapshot)
	}
	slices.SortFunc(snapshots, func(a, b Snapshot) int {
		if order := a.TakenAt.Compare(b.TakenAt); order != 0 {
			return order
		}
		return slices.Compare(a.ID[:], b.ID[:])
	})
	return snapshots
}
