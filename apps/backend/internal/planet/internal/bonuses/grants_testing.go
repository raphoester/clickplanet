//go:build testing

package bonuses

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var _ Grants = (*MemoryGrants)(nil)

type MemoryGrants struct {
	mu       sync.Mutex
	recorded *cpcolls.Set[GrantID]
}

func NewMemoryGrants() *MemoryGrants {
	return &MemoryGrants{recorded: cpcolls.NewSet[GrantID]()}
}

func (m *MemoryGrants) Record(_ context.Context, grant GrantID, _ time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.recorded.Contains(grant) {
		return false, nil
	}
	m.recorded.Add(grant)
	return true, nil
}

func (m *MemoryGrants) Recorded(_ context.Context, grant GrantID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.recorded.Contains(grant), nil
}
