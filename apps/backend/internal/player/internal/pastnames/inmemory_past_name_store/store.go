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
}

var _ pastnames.Storage = (*Store)(nil)

func (s *Store) Record(_ context.Context, name pastnames.PastName) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := name.TakeSnapshot()
	if slices.ContainsFunc(s.kept, func(was pastnames.Snapshot) bool {
		return was.Account == kept.Account && was.LeftAt.Equal(kept.LeftAt)
	}) {
		return nil
	}
	s.kept = append(s.kept, kept)
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
	s.kept = slices.DeleteFunc(s.kept, func(kept pastnames.Snapshot) bool { return kept.Account == account })
	return nil
}
