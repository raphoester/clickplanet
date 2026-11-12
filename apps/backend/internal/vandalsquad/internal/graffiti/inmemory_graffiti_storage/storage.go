//go:build testing

package inmemory_graffiti_storage

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []graffiti.Snapshot
	err  error
}

var _ graffiti.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Kept() []graffiti.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.firstFlaggedFirst(func(graffiti.Snapshot) bool { return true })
}

func (s *Storage) Save(_ context.Context, saved ...graffiti.Graffiti) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, flagged := range saved {
		snapshot := flagged.TakeSnapshot()
		s.kept = slices.DeleteFunc(s.kept, func(kept graffiti.Snapshot) bool { return kept.ID == snapshot.ID })
		s.kept = append(s.kept, snapshot)
	}
	return nil
}

func (s *Storage) PaintedBy(_ context.Context, account cpsession.AccountID) ([]graffiti.Graffiti, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}
	painted := s.firstFlaggedFirst(func(kept graffiti.Snapshot) bool {
		return account != cpsession.NoAccount && kept.Account == account
	})
	found := make([]graffiti.Graffiti, 0, len(painted))
	for _, snapshot := range painted {
		restored, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore a graffiti: %w", err)
		}
		found = append(found, restored)
	}
	return found, nil
}

func (s *Storage) firstFlaggedFirst(wanted func(graffiti.Snapshot) bool) []graffiti.Snapshot {
	var found []graffiti.Snapshot
	for _, kept := range s.kept {
		if wanted(kept) {
			found = append(found, kept)
		}
	}
	slices.SortStableFunc(found, func(a, b graffiti.Snapshot) int { return a.FlaggedAt.Compare(b.FlaggedAt) })
	return found
}
