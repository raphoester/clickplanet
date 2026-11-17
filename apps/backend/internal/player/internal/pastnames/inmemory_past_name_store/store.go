//go:build testing

package inmemory_past_name_store

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func New() *Store {
	return &Store{}
}

type Store struct {
	mu   sync.Mutex
	kept []pastnames.Snapshot
	err  error
}

var _ pastnames.Storage = (*Store)(nil)

// FailWith makes every call answer err.
func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *Store) Record(_ context.Context, name pastnames.PastName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.kept = append(s.kept, name.TakeSnapshot())
	return nil
}

func (s *Store) Kept() []pastnames.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.kept)
}

func (s *Store) Forget(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.kept = slices.DeleteFunc(s.kept, func(kept pastnames.Snapshot) bool { return kept.Account == account })
	return nil
}
