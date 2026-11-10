//go:build testing

package jury

import (
	"slices"
	"sync"
	"time"
)

var _ Turns = (*MemoryTurns)(nil)

type Forgotten struct {
	Account string
	Until   time.Time
}

type MemoryTurns struct {
	mu        sync.Mutex
	forgotten []Forgotten
}

func (m *MemoryTurns) ForgetTurns(account string, until time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.forgotten = append(m.forgotten, Forgotten{Account: account, Until: until})
}

func (m *MemoryTurns) Forgotten() []Forgotten {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.forgotten)
}
