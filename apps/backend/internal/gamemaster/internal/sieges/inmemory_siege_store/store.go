//go:build testing

package inmemory_siege_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var _ sieges.Store = (*Store)(nil)

func New() *Store {
	return &Store{modes: map[moments.RoundID]moments.Mode{}, closed: cpcolls.NewSet[moments.RoundID]()}
}

// Store keeps the sieges in maps.
type Store struct {
	mu       sync.Mutex
	modes    map[moments.RoundID]moments.Mode
	closed   *cpcolls.Set[moments.RoundID]
	failWith error
}

func (s *Store) KeepJudged(_ context.Context, judgement moments.Judgement) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ok := s.modes[judgement.Round]; ok {
		return false, nil
	}
	s.modes[judgement.Round] = judgement.Mode
	return true, nil
}

func (s *Store) Judged(context.Context) ([]moments.Judgement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	judged := make([]moments.Judgement, 0, len(s.modes))
	for round, mode := range s.modes {
		judged = append(judged, moments.Judgement{Round: round, Mode: mode})
	}
	return judged, nil
}

func (s *Store) KeepClosed(_ context.Context, arena moments.RoundID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if s.closed.Contains(arena) {
		return false, nil
	}
	s.closed.Add(arena)
	return true, nil
}

func (s *Store) Closed(context.Context) ([]moments.RoundID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	closed := make([]moments.RoundID, 0, s.closed.Len())
	s.closed.ForEach(func(arena moments.RoundID) { closed = append(closed, arena) })
	return closed, nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
