//go:build testing

package judgements

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var _ Store = (*MemoryStore)(nil)

// MemoryStore judges the submissions it is told wait, in the order told.
type MemoryStore struct {
	mu      sync.Mutex
	waiting []submissions.SubmissionID
	rows    map[submissions.SubmissionID]Snapshot
}

func NewMemoryStore(waiting ...submissions.SubmissionID) *MemoryStore {
	return &MemoryStore{waiting: waiting, rows: map[submissions.SubmissionID]Snapshot{}}
}

func (m *MemoryStore) Unjudged(_ context.Context, limit int) ([]submissions.SubmissionID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var unjudged []submissions.SubmissionID
	for _, id := range m.waiting {
		if _, ok := m.rows[id]; !ok && len(unjudged) < limit {
			unjudged = append(unjudged, id)
		}
	}
	return unjudged, nil
}

func (m *MemoryStore) Save(_ context.Context, judgement Judgement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := judgement.TakeSnapshot()
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
