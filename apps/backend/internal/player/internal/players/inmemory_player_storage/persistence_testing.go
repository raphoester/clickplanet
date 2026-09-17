//go:build testing

package inmemory_player_storage

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// MemoryPersistence is players.Persistence in maps, which can fail on demand.
type MemoryPersistence struct {
	mu       sync.Mutex
	profiles map[players.AccountID]players.Profile
	stats    map[players.AccountID]players.Stats
	saves    int
	failWith error
}

var _ players.Persistence = (*MemoryPersistence)(nil)

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{profiles: map[players.AccountID]players.Profile{}, stats: map[players.AccountID]players.Stats{}}
}

func (m *MemoryPersistence) Load(context.Context) (players.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return players.Snapshot{}, m.failWith
	}
	return players.Snapshot{
		Profiles: slices.Collect(maps.Values(m.profiles)),
		Stats:    slices.Collect(maps.Values(m.stats)),
	}, nil
}

func (m *MemoryPersistence) Save(_ context.Context, changes players.Changes) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	m.saves++
	for _, profile := range changes.Profiles {
		m.profiles[profile.Account] = profile
	}
	for _, stats := range changes.Stats {
		m.stats[stats.Account] = stats
	}
	for _, account := range changes.DeletedProfiles {
		delete(m.profiles, account)
	}
	for _, account := range changes.DeletedStats {
		delete(m.stats, account)
	}
	return nil
}

// Saves is how many saves were written.
func (m *MemoryPersistence) Saves() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saves
}

func (m *MemoryPersistence) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failWith = err
}

func (m *MemoryPersistence) Heal() {
	m.FailWith(nil)
}
