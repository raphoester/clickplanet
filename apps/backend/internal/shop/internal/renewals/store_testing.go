//go:build testing

package renewals

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[SubscriptionID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[SubscriptionID]Snapshot{}}
}

func (m *MemoryStore) Seed(rows ...Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range rows {
		m.rows[row.ID] = row
	}
}

func (m *MemoryStore) Renewal(_ context.Context, id SubscriptionID) (Renewal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Renewal{}, fmt.Errorf("%s: %w", id, ErrUnknownRenewal)
	}
	return row.Restore()
}

func (m *MemoryStore) Bought(_ context.Context, buyer orders.AccountID) ([]Renewal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rows []Snapshot
	for _, row := range m.rows {
		if row.Order.Buyer == buyer {
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b Snapshot) int {
		return cmp.Or(a.PeriodEndsAt.Compare(b.PeriodEndsAt), cmp.Compare(a.ID, b.ID))
	})
	bought := make([]Renewal, 0, len(rows))
	for _, row := range rows {
		renewal, err := row.Restore()
		if err != nil {
			return nil, err
		}
		bought = append(bought, renewal)
	}
	return bought, nil
}

func (m *MemoryStore) Save(_ context.Context, renewal Renewal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := renewal.TakeSnapshot()
	m.rows[row.ID] = row
	return nil
}

// Kept is the renewal kept under the id, and whether one is.
func (m *MemoryStore) Kept(id SubscriptionID) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	return row, ok
}
