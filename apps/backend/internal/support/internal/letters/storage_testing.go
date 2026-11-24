//go:build testing

package letters

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var _ Storage = (*MemoryStorage)(nil)

type MemoryStorage struct {
	mu   sync.Mutex
	rows map[LetterID]Snapshot
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{rows: map[LetterID]Snapshot{}}
}

func (m *MemoryStorage) Seed(rows ...Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, row := range rows {
		m.rows[row.ID] = row
	}
}

// Kept is every letter, the first written first.
func (m *MemoryStorage) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sorted(func(Snapshot) bool { return true })
}

func (m *MemoryStorage) sorted(keep func(Snapshot) bool) []Snapshot {
	var rows []Snapshot
	for _, row := range m.rows {
		if keep(row) {
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b Snapshot) int {
		if c := a.Sent[0].At.Compare(b.Sent[0].At); c != 0 {
			return c
		}
		return slices.Compare(a.ID[:], b.ID[:])
	})
	return rows
}

func (m *MemoryStorage) Letter(_ context.Context, id LetterID) (Letter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Letter{}, fmt.Errorf("%s: %w", id, ErrUnknownLetter)
	}
	return row.Restore()
}

func (m *MemoryStorage) To(_ context.Context, account notes.AccountID) ([]Letter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return restored(m.sorted(func(row Snapshot) bool { return row.Account == account && account != notes.AccountID{} }))
}

func (m *MemoryStorage) About(_ context.Context, submission SubmissionID) ([]Letter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return restored(m.sorted(func(row Snapshot) bool { return row.Submission == submission }))
}

func (m *MemoryStorage) WrittenBefore(_ context.Context, at time.Time) ([]Letter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return restored(m.sorted(func(row Snapshot) bool {
		return row.Account != notes.AccountID{} && row.Sent[0].At.Before(at)
	}))
}

func restored(rows []Snapshot) ([]Letter, error) {
	found := make([]Letter, 0, len(rows))
	for _, row := range rows {
		letter, err := row.Restore()
		if err != nil {
			return nil, err
		}
		found = append(found, letter)
	}
	return found, nil
}

func (m *MemoryStorage) Save(_ context.Context, saved ...Letter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, letter := range saved {
		row := letter.TakeSnapshot()
		for id, kept := range m.rows {
			if id != row.ID && kept.Submission == row.Submission {
				return fmt.Errorf("%w: %s", ErrWrittenAlready, row.Submission)
			}
		}
	}
	for _, letter := range saved {
		row := letter.TakeSnapshot()
		m.rows[row.ID] = row
	}
	return nil
}

// SequentialIDs draws 00000000-0000-0000-0000-000000000001, then 2, and so on.
type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (LetterID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return LetterID{15: s.next}, nil
}

// TestWritten is a letter to the account, its first mail sent at a time, about the submission of the same id.
func TestWritten(id LetterID, account notes.AccountID, at time.Time) Snapshot {
	return Snapshot{
		ID: id, Submission: SubmissionID(id), Account: account, To: "ada@example.com", Subject: "Your flag Catalonia",
		Sent: []Sent{{Text: "Which city is its capital?", Mail: "l1@clickplanet.lol", At: at}},
	}
}

func Restored(snapshot Snapshot) Letter {
	letter, err := snapshot.Restore()
	if err != nil {
		panic(err)
	}
	return letter
}
