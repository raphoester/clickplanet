//go:build testing

package inmemory_grant_store

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func New() *Store {
	return &Store{kept: map[grants.GrantID]grants.Snapshot{}}
}

type Store struct {
	mu   sync.Mutex
	kept map[grants.GrantID]grants.Snapshot
	err  error
}

var _ grants.Store = (*Store)(nil)

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

// Every grant kept, by id.
func (s *Store) Kept() []grants.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := make([]grants.Snapshot, 0, len(s.kept))
	for _, snapshot := range s.kept {
		kept = append(kept, snapshot)
	}
	slices.SortFunc(kept, func(a, b grants.Snapshot) int { return slices.Compare(a.ID[:], b.ID[:]) })
	return kept
}

// Kept once the fake transaction commits, as postgres would.
func (s *Store) SaveGrant(ctx context.Context, grant grants.Grant) error {
	s.mu.Lock()
	err := s.err
	s.mu.Unlock()
	if err != nil {
		return err
	}
	snapshot := grant.TakeSnapshot()
	cpatomic.OnCommit(ctx, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.kept[snapshot.ID] = snapshot
	})
	return nil
}

func (s *Store) Grant(_ context.Context, id grants.GrantID) (grants.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return grants.Grant{}, s.err
	}
	snapshot, found := s.kept[id]
	if !found {
		return grants.Grant{}, grants.ErrGrantNotFound
	}
	return snapshot.Restore() //nolint:wrapcheck // a test store hands the rule's error on.
}

func (s *Store) PruneGrants(_ context.Context, expiredBy time.Time, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return 0, s.err
	}
	pruned := 0
	for id, snapshot := range s.kept {
		if pruned == limit {
			break
		}
		if !snapshot.ExpiresAt.After(expiredBy) {
			delete(s.kept, id)
			pruned++
		}
	}
	return pruned, nil
}
