//go:build testing

package inmemory_grant_store

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Store struct {
	mu     sync.Mutex
	grants map[books.GrantID]books.Grant
}

var _ books.Grants = (*Store)(nil)

func New() *Store {
	return &Store{grants: map[books.GrantID]books.Grant{}}
}

func (s *Store) Keep(_ context.Context, grant books.Grant) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, kept := s.grants[grant.ID]; kept {
		return false, nil
	}
	s.grants[grant.ID] = grant
	return true, nil
}

func (s *Store) Since(_ context.Context, at time.Time) ([]books.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var since []books.Grant
	for _, grant := range s.grants {
		if !grant.At.Before(at) {
			since = append(since, grant)
		}
	}
	slices.SortFunc(since, func(a, b books.Grant) int { return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.ID, b.ID)) })
	return since, nil
}
