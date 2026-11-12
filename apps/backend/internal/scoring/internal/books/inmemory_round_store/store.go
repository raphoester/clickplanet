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
	start        books.Start
	opened       int
	countedBelow books.TakeMark
	countries    map[books.Country]uint64
	players      map[books.AccountID]map[books.Country]uint64
}

type Store struct {
	mu     sync.Mutex
	rounds map[books.RoundID]*kept
	missed map[books.RoundID]books.MissedRound
}

var (
	_ books.Rounds = (*Store)(nil)
	_ books.Takes  = (*Store)(nil)
)

func New() *Store {
	return &Store{rounds: map[books.RoundID]*kept{}, missed: map[books.RoundID]books.MissedRound{}}
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

func (s *Store) Open(_ context.Context, round books.Round, start books.Start) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.rounds[round.ID]; found {
		return false, nil
	}
	kept := &kept{
		scored: books.Scored{Round: round}, start: books.Start{}, opened: len(s.rounds),
		countries: map[books.Country]uint64{}, players: map[books.AccountID]map[books.Country]uint64{},
	}
	for country, tiles := range start {
		if tiles > 0 {
			kept.start[country] = tiles
		}
	}
	s.rounds[round.ID] = kept
	return true, nil
}

func (s *Store) StartOf(_ context.Context, id books.RoundID) (books.Start, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, found := s.rounds[id]
	if !found {
		return books.Start{}, nil
	}
	return maps.Clone(at.start), nil
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

// Fights is the tiles each country took in the round, as Count counted them.
func (s *Store) Fights(_ context.Context, id books.RoundID) (map[books.Country]uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fights := map[books.Country]uint32{}
	if at, found := s.rounds[id]; found {
		for country, tiles := range at.countries {
			fights[country] = uint32(tiles) //nolint:gosec // a country takes a map's tiles at most.
		}
	}
	return fights, nil
}

func (s *Store) Lock(context.Context, books.RoundID) error {
	return nil
}

func (s *Store) Missed(_ context.Context, id books.RoundID) (books.MissedRound, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	missed, found := s.missed[id]
	return missed, found, nil
}

func (s *Store) KeepMissed(_ context.Context, missed books.MissedRound) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, kept := s.missed[missed.ID]; !kept {
		s.missed[missed.ID] = missed
	}
	return nil
}

func (s *Store) Count(_ context.Context, held map[books.RoundID][]books.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, takes := range held {
		at, found := s.rounds[id]
		if !found {
			continue
		}
		var uncounted []books.Take
		uncounted, at.countedBelow = at.countedBelow.Uncounted(takes)
		for _, take := range uncounted {
			at.countries[take.Country]++
			if at.players[take.Account] == nil {
				at.players[take.Account] = map[books.Country]uint64{}
			}
			at.players[take.Account][take.Country]++
		}
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
