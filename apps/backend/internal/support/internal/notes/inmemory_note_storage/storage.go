//go:build testing

package inmemory_note_storage

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []notes.Snapshot
	err  error
}

var _ notes.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Seed(snapshots ...notes.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, snapshots...)
}

func (s *Storage) Kept() []notes.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.oldestFirst(func(notes.Snapshot) bool { return true })
}

func (s *Storage) Save(_ context.Context, saved ...notes.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, note := range saved {
		snapshot := note.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept notes.Snapshot) bool { return kept.ID == snapshot.ID })
		s.kept = append(s.kept, snapshot)
	}
	return nil
}

func (s *Storage) SentBy(_ context.Context, account notes.AccountID) ([]notes.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	return restored(s.oldestFirst(func(kept notes.Snapshot) bool {
		return kept.Account == account && account != notes.AccountID{}
	}))
}

func (s *Storage) WithSenderBefore(_ context.Context, cutoff time.Time, limit int) ([]notes.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	old := s.oldestFirst(func(kept notes.Snapshot) bool {
		return kept.At.Before(cutoff) && kept.Account != notes.AccountID{}
	})
	return restored(old[:min(len(old), limit)])
}

func (s *Storage) oldestFirst(wanted func(notes.Snapshot) bool) []notes.Snapshot {
	var found []notes.Snapshot
	for _, kept := range s.kept {
		if wanted(kept) {
			found = append(found, kept)
		}
	}
	slices.SortStableFunc(found, func(a, b notes.Snapshot) int { return a.At.Compare(b.At) })
	return found
}

func restored(snapshots []notes.Snapshot) ([]notes.Note, error) {
	found := make([]notes.Note, 0, len(snapshots))
	for _, snapshot := range snapshots {
		note, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a note: %w", err)
		}
		found = append(found, note)
	}
	return found, nil
}
