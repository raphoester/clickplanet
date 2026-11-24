//go:build testing

package grants

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[GrantID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[GrantID]Snapshot{}}
}

func (m *MemoryStore) Grant(_ context.Context, id GrantID) (Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Unheard(id), nil
	}
	return row.Restore()
}

func (m *MemoryStore) Held(_ context.Context, submission submissions.SubmissionID) ([]Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var held []Snapshot
	for _, row := range m.rows {
		if row.Held && row.Submission == submission && !row.Granted && !row.TakenBack {
			held = append(held, row)
		}
	}
	slices.SortFunc(held, func(a, b Snapshot) int {
		if order := a.HeldAt.Compare(b.HeldAt); order != 0 {
			return order
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	grants := make([]Grant, 0, len(held))
	for _, row := range held {
		grant, err := row.Restore()
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

// Save keeps what postgres keeps: a hold once, a grant once, and its taking back once.
func (m *MemoryStore) Save(_ context.Context, grant Grant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := grant.TakeSnapshot()
	kept := m.rows[row.ID]
	kept.ID = row.ID
	if row.Held && !kept.Held {
		kept.Held, kept.Submission, kept.Period, kept.HeldAt = true, row.Submission, row.Period, row.HeldAt
	}
	if row.Granted && !kept.Granted {
		kept.Granted, kept.Flag, kept.Added, kept.GrantedAt = true, row.Flag, row.Added, row.GrantedAt
	}
	if row.TakenBack && !kept.TakenBack {
		kept.TakenBack, kept.TakenBackAt = true, row.TakenBackAt
	}
	m.rows[row.ID] = kept
	return nil
}
