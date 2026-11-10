//go:build testing

package inmemory_arena_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var _ arenas.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		modes: map[moments.RoundID]moments.Mode{}, opened: cpcolls.NewSet[moments.RoundID](),
		closed: cpcolls.NewSet[moments.RoundID](),
	}
}

// Store keeps the arena battles in maps.
type Store struct {
	mu       sync.Mutex
	modes    map[moments.RoundID]moments.Mode
	opened   *cpcolls.Set[moments.RoundID]
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

func (s *Store) KeepOpened(_ context.Context, arena moments.RoundID) (bool, error) {
	return s.keep(s.opened, arena)
}

func (s *Store) KeepClosed(_ context.Context, arena moments.RoundID) (bool, error) {
	return s.keep(s.closed, arena)
}

func (s *Store) keep(in *cpcolls.Set[moments.RoundID], arena moments.RoundID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if in.Contains(arena) {
		return false, nil
	}
	in.Add(arena)
	return true, nil
}

func (s *Store) ArenasKept(context.Context) (moments.ArenasKept, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return moments.ArenasKept{}, s.failWith
	}
	var kept moments.ArenasKept
	for round, mode := range s.modes {
		kept.Judged = append(kept.Judged, moments.Judgement{Round: round, Mode: mode})
	}
	s.opened.ForEach(func(arena moments.RoundID) { kept.Opened = append(kept.Opened, arena) })
	s.closed.ForEach(func(arena moments.RoundID) { kept.Closed = append(kept.Closed, arena) })
	return kept, nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
