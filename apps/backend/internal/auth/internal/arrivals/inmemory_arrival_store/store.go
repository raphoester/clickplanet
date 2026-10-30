//go:build testing

package inmemory_arrival_store

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

func New() *Store {
	return &Store{}
}

type Store struct {
	mu   sync.Mutex
	kept []arrivals.Snapshot
	err  error
}

var _ arrivals.Store = (*Store)(nil)

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Store) Seed(snapshots ...arrivals.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, snapshots...)
}

func (s *Store) Kept() []arrivals.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.oldestFirst(func(arrivals.Snapshot) bool { return true })
}

func (s *Store) Save(_ context.Context, saved ...arrivals.Arrival) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, arrival := range saved {
		snapshot := arrival.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept arrivals.Snapshot) bool { return kept.ID == snapshot.ID })
		s.kept = append(s.kept, snapshot)
	}
	return nil
}

func (s *Store) Of(_ context.Context, account arrivals.AccountID) ([]arrivals.Arrival, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	found := []arrivals.Arrival{}
	for _, snapshot := range s.oldestFirst(func(kept arrivals.Snapshot) bool {
		return kept.Account == account && account != arrivals.AccountID{}
	}) {
		arrival, err := snapshot.Restore()
		if err != nil {
			return nil, err //nolint:wrapcheck // a test store hands the rule's error on.
		}
		found = append(found, arrival)
	}
	return found, nil
}

func (s *Store) oldestFirst(keep func(arrivals.Snapshot) bool) []arrivals.Snapshot {
	var picked []arrivals.Snapshot
	for _, snapshot := range s.kept {
		if keep(snapshot) {
			picked = append(picked, snapshot)
		}
	}
	slices.SortStableFunc(picked, func(a, b arrivals.Snapshot) int {
		return cmp.Or(a.At.Compare(b.At), slices.Compare(a.ID[:], b.ID[:]))
	})
	return picked
}
