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
		judged: map[moments.RoundID]moments.Judgement{}, opened: cpcolls.NewSet[moments.RoundID](),
		closed: cpcolls.NewSet[moments.RoundID](),
	}
}

// Store keeps the arena battles in maps.
type Store struct {
	mu     sync.Mutex
	judged map[moments.RoundID]moments.Judgement
	opened *cpcolls.Set[moments.RoundID]
	closed *cpcolls.Set[moments.RoundID]
}

func (s *Store) KeepJudged(_ context.Context, judgement moments.Judgement) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.judged[judgement.Round]; ok {
		return false, nil
	}
	s.judged[judgement.Round] = judgement
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

	if in.Contains(arena) {
		return false, nil
	}
	in.Add(arena)
	return true, nil
}

func (s *Store) ArenasKept(context.Context) (moments.ArenasKept, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var kept moments.ArenasKept
	for _, judgement := range s.judged {
		kept.Judged = append(kept.Judged, judgement)
	}
	s.opened.ForEach(func(arena moments.RoundID) { kept.Opened = append(kept.Opened, arena) })
	s.closed.ForEach(func(arena moments.RoundID) { kept.Closed = append(kept.Closed, arena) })
	return kept, nil
}
