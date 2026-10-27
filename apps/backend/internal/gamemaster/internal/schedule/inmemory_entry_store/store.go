//go:build testing

package inmemory_entry_store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Store struct {
	mu       sync.Mutex
	entries  map[schedule.EntryID]schedule.EntrySnapshot
	failWith error
}

var _ schedule.Store = (*Store)(nil)

func New() *Store {
	return &Store{entries: map[schedule.EntryID]schedule.EntrySnapshot{}}
}

func (s *Store) Entry(_ context.Context, id schedule.EntryID) (schedule.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return schedule.Entry{}, s.failWith
	}
	entry, ok := s.entries[id]
	if !ok {
		return schedule.Entry{}, fmt.Errorf("%s: %w", id, schedule.ErrUnknownEntry)
	}
	return entry.Restore(), nil
}

func (s *Store) Unended(_ context.Context, at time.Time) ([]schedule.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var unended []schedule.Entry
	for _, entry := range s.entries {
		if entry.EndsAt.After(at) {
			unended = append(unended, entry.Restore())
		}
	}
	return unended, nil
}

func (s *Store) Save(_ context.Context, entry schedule.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	snapshot := entry.TakeSnapshot()
	s.entries[snapshot.ID] = snapshot
	return nil
}

func (s *Store) Delete(_ context.Context, id schedule.EntryID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.entries, id)
	return nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
