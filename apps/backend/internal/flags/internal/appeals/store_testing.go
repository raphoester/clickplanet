//go:build testing

package appeals

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[submissions.SubmissionID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[submissions.SubmissionID]Snapshot{}}
}

func (m *MemoryStore) Of(_ context.Context, submission submissions.SubmissionID) ([]Appeal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[submission]
	if !ok {
		return nil, nil
	}
	appeal, err := row.Restore()
	if err != nil {
		return nil, err
	}
	return []Appeal{appeal}, nil
}

func (m *MemoryStore) Save(_ context.Context, appeal Appeal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := appeal.TakeSnapshot()
	m.rows[row.Submission] = row
	return nil
}

func (m *MemoryStore) Kept() map[submissions.SubmissionID]Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := make(map[submissions.SubmissionID]Snapshot, len(m.rows))
	for id, row := range m.rows {
		kept[id] = row
	}
	return kept
}
