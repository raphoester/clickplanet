//go:build testing

package notices

import (
	"context"
	"sync"
	"time"
)

var _ Inbox = (*MemoryInbox)(nil)

type MemoryInbox struct {
	mu   sync.Mutex
	kept map[EventID]time.Time
}

func NewMemoryInbox() *MemoryInbox {
	return &MemoryInbox{kept: map[EventID]time.Time{}}
}

func (m *MemoryInbox) Keep(_ context.Context, id EventID, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.kept[id]; ok {
		return false, nil
	}
	m.kept[id] = at
	return true, nil
}
