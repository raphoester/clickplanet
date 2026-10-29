//go:build testing

package inmemory_moment_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var _ moments.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		rounds: map[moments.RoundID]moments.ScoredRound{}, books: map[moments.BookID]moments.ClosedBook{},
		opened: cpcolls.NewSet[moments.RoundID](), told: map[moments.RoundID]time.Time{},
	}
}

type Store struct {
	mu       sync.Mutex
	rounds   map[moments.RoundID]moments.ScoredRound
	books    map[moments.BookID]moments.ClosedBook
	opened   *cpcolls.Set[moments.RoundID]
	told     map[moments.RoundID]time.Time
	failWith error
}

func (s *Store) KeepRound(_ context.Context, scored moments.ScoredRound) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ok := s.rounds[scored.Round]; ok {
		return false, nil
	}
	s.rounds[scored.Round] = scored
	return true, nil
}

func (s *Store) KeepBook(_ context.Context, closed moments.ClosedBook) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ok := s.books[closed.Book]; ok {
		return false, nil
	}
	s.books[closed.Book] = closed
	return true, nil
}

func (s *Store) KeepOpened(_ context.Context, round moments.RoundID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.opened.Add(round)
	return nil
}

func (s *Store) KeepTold(_ context.Context, day moments.ToldDay) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if told, ok := s.told[day.Round]; ok && told.Equal(day.EndsAt) {
		return false, nil
	}
	s.told[day.Round] = day.EndsAt
	return true, nil
}

func (s *Store) Opened(context.Context) ([]moments.RoundID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	opened := make([]moments.RoundID, 0, s.opened.Len())
	s.opened.ForEach(func(round moments.RoundID) { opened = append(opened, round) })
	return opened, nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
