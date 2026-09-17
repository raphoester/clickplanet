//go:build testing

package inmemory_message_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// MemoryPersistence is a Persistence held in a slice, for tests that need a storage but not postgres.
type MemoryPersistence struct {
	mu      sync.Mutex
	rows    []messages.Record
	failing error
}

func NewMemoryPersistence(rows ...messages.Record) *MemoryPersistence {
	return &MemoryPersistence{rows: slices.Clone(rows)}
}

func (m *MemoryPersistence) Insert(_ context.Context, record messages.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.rows = append(m.rows, record)
	return nil
}

func (m *MemoryPersistence) Recent(_ context.Context, since time.Time, limit int) ([]messages.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var recent []messages.Message
	for _, row := range m.rows {
		if !row.Message.SentAt.Before(since) {
			recent = append(recent, row.Message)
		}
	}
	return recent[max(0, len(recent)-limit):], nil
}

func (m *MemoryPersistence) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return 0, m.failing
	}
	before := len(m.rows)
	m.rows = slices.DeleteFunc(m.rows, func(row messages.Record) bool { return row.Message.SentAt.Before(cutoff) })
	return int64(before - len(m.rows)), nil
}

func (m *MemoryPersistence) Stored() []messages.Record {
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
