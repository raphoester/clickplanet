//go:build testing

package notices

import (
	"context"
	"slices"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[string]Notice
	failing error
}

func NewMemoryStore(notices ...Notice) *MemoryStore {
	rows := make(map[string]Notice, len(notices))
	for _, notice := range notices {
		rows[notice.Account] = notice
	}
	return &MemoryStore{rows: rows}
}

func (m *MemoryStore) Noticed(_ context.Context, account string) (Notice, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Notice{}, false, m.failing
	}
	notice, ok := m.rows[account]
	return notice, ok, nil
}

func (m *MemoryStore) Keep(_ context.Context, notice Notice) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return false, m.failing
	}
	if _, noticed := m.rows[notice.Account]; noticed {
		return false, nil
	}
	m.rows[notice.Account] = notice
	return true, nil
}

func (m *MemoryStore) Acknowledge(_ context.Context, account string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	notice, noticed := m.rows[account]
	if !noticed || !notice.Pending() {
		return nil
	}
	notice.AcknowledgedAt = at
	m.rows[account] = notice
	return nil
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

var _ Feed = (*MemoryFeed)(nil)

type MemoryFeed struct {
	mu        sync.Mutex
	published []Notice
}

func (m *MemoryFeed) Publish(notice Notice) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.published = append(m.published, notice)
}

func (m *MemoryFeed) Published() []Notice {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.published)
}
