//go:build testing

package inmemory_ban_storage

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

// MemoryPersistence is a Persistence held in a slice, for tests that need a storage but not postgres.
type MemoryPersistence struct {
	mu      sync.Mutex
	rows    []bans.Ban
	failing error
}

func NewMemoryPersistence(rows ...bans.Ban) *MemoryPersistence {
	return &MemoryPersistence{rows: slices.Clone(rows)}
}

func (m *MemoryPersistence) All(_ context.Context) ([]bans.Ban, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	return slices.Clone(m.rows), nil
}

func (m *MemoryPersistence) Upsert(_ context.Context, ban bans.Ban) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}

	at := slices.IndexFunc(m.rows, func(row bans.Ban) bool { return row.AuthorTag == ban.AuthorTag })
	if at == -1 {
		m.rows = append(m.rows, ban)
		return nil
	}
	m.rows[at] = ban
	return nil
}

func (m *MemoryPersistence) Delete(_ context.Context, tag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.rows = slices.DeleteFunc(m.rows, func(row bans.Ban) bool { return row.AuthorTag == tag })
	return nil
}

func (m *MemoryPersistence) Stored() []bans.Ban {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.rows)
}

// FailWith makes every call return err until Heal.
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
