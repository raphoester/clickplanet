//go:build testing

package submissions

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[SubmissionID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[SubmissionID]Snapshot{}}
}

func (m *MemoryStore) Seed(rows ...Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range rows {
		m.rows[row.ID] = row
	}
}

func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sorted(func(Snapshot) bool { return true })
}

func (m *MemoryStore) sorted(keep func(Snapshot) bool) []Snapshot {
	var rows []Snapshot
	for _, row := range m.rows {
		if keep(row) {
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b Snapshot) int {
		if c := b.SubmittedAt.Compare(a.SubmittedAt); c != 0 {
			return c
		}
		return slices.Compare(a.ID[:], b.ID[:])
	})
	return rows
}

func (m *MemoryStore) Submission(_ context.Context, id SubmissionID) (Submission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Submission{}, fmt.Errorf("%s: %w", id, ErrUnknownSubmission)
	}
	return row.Restore()
}

func (m *MemoryStore) Founded(_ context.Context, account AccountID) ([]Submission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return restored(m.sorted(func(row Snapshot) bool { return row.Account == account }))
}

func (m *MemoryStore) RejectedHeld(context.Context) ([]Submission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return restored(m.sorted(func(row Snapshot) bool { return row.State == Rejected && row.Held }))
}

func restored(rows []Snapshot) ([]Submission, error) {
	submissions := make([]Submission, 0, len(rows))
	for _, row := range rows {
		submission, err := row.Restore()
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, submission)
	}
	return submissions, nil
}

func (m *MemoryStore) Save(_ context.Context, submissions ...Submission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, submission := range submissions {
		row := submission.TakeSnapshot()
		m.rows[row.ID] = row
	}
	return nil
}

// Sequence draws 00000000-0000-0000-0000-000000000001, then 2, and so on.
type Sequence struct {
	mu   sync.Mutex
	next byte
}

func (s *Sequence) NewID() (SubmissionID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return SubmissionID{15: s.next}, nil
}
