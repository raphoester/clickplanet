//go:build testing

package checkouts

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows []Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Save(_ context.Context, checkout Checkout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows = append(m.rows, checkout.TakeSnapshot())
	return nil
}

func (m *MemoryStore) OfTarget(_ context.Context, target orders.Target) ([]Checkout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var opened []Checkout
	for _, row := range m.rows {
		if row.Target != target {
			continue
		}
		checkout, err := row.Restore()
		if err != nil {
			return nil, err
		}
		opened = append(opened, checkout)
	}
	return opened, nil
}
