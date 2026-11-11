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
		missedRounds: map[moments.RoundID]moments.MissedRound{}, missedBooks: map[moments.BookID]moments.MissedBook{},
		opened: cpcolls.NewSet[moments.RoundID](), reset: cpcolls.NewSet[moments.BookID](),
		toldSeasons: cpcolls.NewSet[moments.BookID](),
		toldDays:    map[moments.RoundID]time.Time{}, toldBattles: cpcolls.NewSet[moments.RoundID](),
	}
}

type Store struct {
	mu           sync.Mutex
	rounds       map[moments.RoundID]moments.ScoredRound
	books        map[moments.BookID]moments.ClosedBook
	missedRounds map[moments.RoundID]moments.MissedRound
	missedBooks  map[moments.BookID]moments.MissedBook
	opened       *cpcolls.Set[moments.RoundID]
	reset        *cpcolls.Set[moments.BookID]
	toldSeasons  *cpcolls.Set[moments.BookID]
	toldDays     map[moments.RoundID]time.Time
	toldBattles  *cpcolls.Set[moments.RoundID]
	failWith     error
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

func (s *Store) KeepMissedRound(_ context.Context, missed moments.MissedRound) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ok := s.missedRounds[missed.Round]; ok {
		return false, nil
	}
	s.missedRounds[missed.Round] = missed
	return true, nil
}

func (s *Store) KeepMissedBook(_ context.Context, missed moments.MissedBook) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ok := s.missedBooks[missed.Book]; ok {
		return false, nil
	}
	s.missedBooks[missed.Book] = missed
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

func (s *Store) KeepReset(_ context.Context, book moments.BookID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if s.reset.Contains(book) {
		return false, nil
	}
	s.reset.Add(book)
	return true, nil
}

func (s *Store) KeepToldSeason(_ context.Context, book moments.BookID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if s.toldSeasons.Contains(book) {
		return false, nil
	}
	s.toldSeasons.Add(book)
	return true, nil
}

func (s *Store) KeepToldDay(_ context.Context, day moments.ToldDay) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if told, ok := s.toldDays[day.Round]; ok && told.Equal(day.EndsAt) {
		return false, nil
	}
	s.toldDays[day.Round] = day.EndsAt
	return true, nil
}

func (s *Store) KeepToldBattle(_ context.Context, round moments.RoundID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if s.toldBattles.Contains(round) {
		return false, nil
	}
	s.toldBattles.Add(round)
	return true, nil
}

func (s *Store) Kept(context.Context) (moments.Kept, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return moments.Kept{}, s.failWith
	}
	var kept moments.Kept
	s.opened.ForEach(func(round moments.RoundID) { kept.Opened = append(kept.Opened, round) })
	for round := range s.rounds {
		kept.Scored = append(kept.Scored, round)
	}
	for round := range s.missedRounds {
		kept.MissedRounds = append(kept.MissedRounds, round)
	}
	for book := range s.books {
		kept.Closed = append(kept.Closed, book)
	}
	for book := range s.missedBooks {
		kept.MissedBooks = append(kept.MissedBooks, book)
	}
	s.reset.ForEach(func(book moments.BookID) { kept.Reset = append(kept.Reset, book) })
	return kept, nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
