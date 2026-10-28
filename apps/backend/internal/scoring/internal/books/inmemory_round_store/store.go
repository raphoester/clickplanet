//go:build testing

package inmemory_round_store

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type kept struct {
	scored       books.Scored
	opened       int
	countedBelow uint64
	countries    map[books.Country]uint64
	players      map[books.AccountID]map[books.Country]uint64
}

type Store struct {
	mu     sync.Mutex
	rounds map[books.RoundID]*kept
}

var (
	_ books.Rounds = (*Store)(nil)
	_ books.Takes  = (*Store)(nil)
)

func New() *Store {
	return &Store{rounds: map[books.RoundID]*kept{}}
}

func (s *Store) Find(_ context.Context, id books.RoundID) (books.Round, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, found := s.rounds[id]
	if !found {
		return books.Round{}, false, nil
	}
	return at.scored.Round, true, nil
}

func (s *Store) Of(_ context.Context, of []books.BookID) ([]books.Round, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var listed []*kept
	for _, at := range s.rounds {
		if slices.Contains(of, at.scored.Round.Book) {
			listed = append(listed, at)
		}
	}
	slices.SortFunc(listed, func(a, b *kept) int { return a.opened - b.opened })
	rounds := make([]books.Round, 0, len(listed))
	for _, at := range listed {
		rounds = append(rounds, at.scored.Round)
	}
	slices.SortStableFunc(rounds, func(a, b books.Round) int { return a.StartsAt.Compare(b.StartsAt) })
	return rounds, nil
}

func (s *Store) Open(_ context.Context, round books.Round) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.rounds[round.ID]; found {
		return false, nil
	}
	s.rounds[round.ID] = &kept{
		scored: books.Scored{Round: round}, opened: len(s.rounds),
		countries: map[books.Country]uint64{}, players: map[books.AccountID]map[books.Country]uint64{},
	}
	return true, nil
}

func (s *Store) Score(_ context.Context, scored books.Scored) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, found := s.rounds[scored.Round.ID]
	if !found || at.scored.Round.Scored {
		return false, nil
	}
	at.scored = books.Scored{Round: scored.Round, Results: append([]books.Result(nil), scored.Results...)}
	return true, nil
}

func (s *Store) ScoredIn(_ context.Context, book books.BookID) ([]books.Scored, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var scored []books.Scored
	for _, at := range s.rounds {
		if at.scored.Round.Book == book && at.scored.Round.Scored {
			scored = append(scored, books.Scored{Round: at.scored.Round, Results: append([]books.Result(nil), at.scored.Results...)})
		}
	}
	slices.SortFunc(scored, func(a, b books.Scored) int {
		if order := a.Round.EndsAt.Compare(b.Round.EndsAt); order != 0 {
			return order
		}
		return a.Round.StartsAt.Compare(b.Round.StartsAt)
	})
	return scored, nil
}

func (s *Store) Count(_ context.Context, rounds []books.RoundID, take books.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range rounds {
		at, found := s.rounds[id]
		if !found || take.Position < at.countedBelow {
			continue
		}
		at.countedBelow = take.Position + 1
		at.countries[take.Country]++
		if at.players[take.Account] == nil {
			at.players[take.Account] = map[books.Country]uint64{}
		}
		at.players[take.Account][take.Country]++
	}
	return nil
}

func (s *Store) ForgetAccount(_ context.Context, account books.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, at := range s.rounds {
		delete(at.players, account)
	}
	return nil
}

func (s *Store) CountedIn(round books.RoundID) books.Counted {
	s.mu.Lock()
	defer s.mu.Unlock()

	counted := books.Counted{Countries: map[books.Country]uint64{}, Players: map[books.AccountID]map[books.Country]uint64{}}
	at, found := s.rounds[round]
	if !found {
		return counted
	}
	maps.Copy(counted.Countries, at.countries)
	for account, countries := range at.players {
		counted.Players[account] = maps.Clone(countries)
	}
	return counted
}
