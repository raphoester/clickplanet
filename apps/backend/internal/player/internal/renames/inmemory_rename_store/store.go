//go:build testing

package inmemory_rename_store

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func New() *Store {
	return &Store{}
}

type Store struct {
	mu   sync.Mutex
	kept []renames.Snapshot
	err  error
}

var _ renames.Storage = (*Store)(nil)

// FailWith makes every call answer err.
func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *Store) Record(_ context.Context, rename renames.Rename) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.kept = append(s.kept, rename.TakeSnapshot())
	return nil
}

func (s *Store) Kept() []renames.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.kept)
}

func (s *Store) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(kept renames.Snapshot) bool { return kept.At.Before(cutoff) })
	return int64(before - len(s.kept)), nil
}

func (s *Store) Forget(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.kept = slices.DeleteFunc(s.kept, func(kept renames.Snapshot) bool { return kept.Account == account })
	for i, kept := range s.kept {
		if kept.By.Account() == account {
			s.kept[i].By = renames.OperatorOf(cpsession.NoAccount, kept.By.Client())
		}
	}
	return nil
}
