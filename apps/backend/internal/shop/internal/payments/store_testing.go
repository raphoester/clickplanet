//go:build testing

package payments

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[PaymentID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[PaymentID]Snapshot{}}
}

func (m *MemoryStore) Payment(_ context.Context, id PaymentID) (Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Payment{}, fmt.Errorf("%s: %w", id, ErrUnknownPayment)
	}
	return row.Restore()
}

func (m *MemoryStore) OfIntent(_ context.Context, intent IntentID) (Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range m.rows {
		if row.Intent == intent {
			return row.Restore()
		}
	}
	return Payment{}, fmt.Errorf("%s: %w", intent, ErrUnknownPayment)
}

func (m *MemoryStore) OfTarget(_ context.Context, target orders.Target) ([]Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rows []Snapshot
	for _, row := range m.rows {
		if row.Order.Target == target {
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b Snapshot) int {
		return cmp.Or(a.PaidAt.Compare(b.PaidAt), cmp.Compare(a.ID, b.ID))
	})
	paid := make([]Payment, 0, len(rows))
	for _, row := range rows {
		payment, err := row.Restore()
		if err != nil {
			return nil, err
		}
		paid = append(paid, payment)
	}
	return paid, nil
}

func (m *MemoryStore) Save(_ context.Context, payment Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := payment.TakeSnapshot()
	if kept, ok := m.rows[row.ID]; ok {
		kept.State, kept.DecidedAt = row.State, row.DecidedAt
		row = kept
	}
	m.rows[row.ID] = row
	return nil
}

// Kept is every payment kept, in no order.
func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := make([]Snapshot, 0, len(m.rows))
	for _, row := range m.rows {
		kept = append(kept, row)
	}
	return kept
}

// TestOrder is a week the account bought for a submission, agreed at a time.
func TestOrder(buyer orders.AccountID, plan plans.Plan, at time.Time) orders.Snapshot {
	target, err := orders.ParseTarget("00000000-0000-0000-0000-000000000001", "")
	if err != nil {
		panic(err)
	}
	return orders.Snapshot{Buyer: buyer, Plan: plan, Target: target, AgreedAt: at}
}
