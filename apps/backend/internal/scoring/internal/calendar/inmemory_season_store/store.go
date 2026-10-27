//go:build testing

package inmemory_season_store

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type Store struct {
	mu        sync.Mutex
	decisions map[calendar.Number]calendar.Decision
	failWith  error
}

var _ calendar.Store = (*Store)(nil)

func New() *Store {
	return &Store{decisions: map[calendar.Number]calendar.Decision{}}
}

func (s *Store) Decision(_ context.Context, number calendar.Number) (calendar.Decision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return calendar.Decision{}, false, s.failWith
	}
	decision, ok := s.decisions[number]
	return decision, ok, nil
}

func (s *Store) Decisions(context.Context) ([]calendar.Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	decisions := make([]calendar.Decision, 0, len(s.decisions))
	for _, decision := range s.decisions {
		decisions = append(decisions, decision)
	}
	slices.SortFunc(decisions, func(a, b calendar.Decision) int { return cmp.Compare(a.Window.Number, b.Window.Number) })
	return decisions, nil
}

func (s *Store) Save(_ context.Context, decision calendar.Decision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.decisions[decision.Window.Number] = decision
	return nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
