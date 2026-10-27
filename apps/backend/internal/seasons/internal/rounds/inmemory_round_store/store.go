//go:build testing

package inmemory_round_store

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

type key struct {
	season calendar.Number
	endsAt int64
	own    bool
}

func keyOf(round rounds.Round) key {
	return key{season: round.Season, endsAt: round.EndsAt.UnixNano(), own: round.Own}
}

type kept struct {
	round        rounds.Round
	closed       bool
	results      []rounds.Result
	countedBelow uint64
	countries    map[rounds.Country]uint64
	players      map[rounds.AccountID]map[rounds.Country]uint64
}

type Store struct {
	mu       sync.Mutex
	rounds   map[key]*kept
	ended    map[calendar.Number]time.Time
	failWith error
}

var (
	_ rounds.Store = (*Store)(nil)
	_ rounds.Takes = (*Store)(nil)
)

func New() *Store {
	return &Store{rounds: map[key]*kept{}, ended: map[calendar.Number]time.Time{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Open(_ context.Context, round rounds.Round) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, known := s.rounds[keyOf(round)]; !known {
		s.rounds[keyOf(round)] = &kept{
			round: round, countries: map[rounds.Country]uint64{}, players: map[rounds.AccountID]map[rounds.Country]uint64{},
		}
	}
	return nil
}

func (s *Store) Unclosed(_ context.Context, endedBy time.Time) ([]rounds.Round, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var ended []rounds.Round
	for _, at := range s.rounds {
		if !at.closed && !at.round.EndsAt.After(endedBy) {
			ended = append(ended, at.round)
		}
	}
	slices.SortFunc(ended, inOrder)
	return ended, nil
}

func inOrder(a, b rounds.Round) int {
	own := func(round rounds.Round) int {
		if round.Own {
			return 1
		}
		return 0
	}
	return cmp.Or(a.EndsAt.Compare(b.EndsAt), cmp.Compare(a.Season, b.Season), cmp.Compare(own(a), own(b)))
}

func (s *Store) FirstEnd(context.Context) (time.Time, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return time.Time{}, false, s.failWith
	}
	var (
		first time.Time
		found bool
	)
	for _, at := range s.rounds {
		if !at.closed && (!found || at.round.EndsAt.Before(first)) {
			first, found = at.round.EndsAt, true
		}
	}
	return first, found, nil
}

func (s *Store) Number(_ context.Context, round rounds.Round) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	return s.numberOf(round), nil
}

func (s *Store) numberOf(round rounds.Round) uint32 {
	number := uint32(1)
	for _, at := range s.rounds {
		if at.round.Season == round.Season && at.round.Own == round.Own && at.round.EndsAt.Before(round.EndsAt) {
			number++
		}
	}
	return number
}

func (s *Store) Close(_ context.Context, round rounds.Round, results []rounds.Result) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	at, known := s.rounds[keyOf(round)]
	if !known || at.closed {
		return false, nil
	}
	at.closed = true
	at.results = slices.Clone(results)
	return true, nil
}

func (s *Store) ClosedIn(_ context.Context, season calendar.Number) ([]rounds.Closed, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var closed []rounds.Closed
	for _, at := range s.rounds {
		if at.round.Season == season && at.closed {
			closed = append(closed, rounds.Closed{Round: at.round, Number: s.numberOf(at.round), Results: append([]rounds.Result(nil), at.results...)})
		}
	}
	slices.SortFunc(closed, func(a, b rounds.Closed) int { return inOrder(a.Round, b.Round) })
	return closed, nil
}

func (s *Store) End(_ context.Context, season calendar.Season) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	if _, ended := s.ended[season.Number]; ended {
		return false, nil
	}
	s.ended[season.Number] = season.EndsAt
	return true, nil
}

func (s *Store) Ended(_ context.Context, season calendar.Number) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return false, s.failWith
	}
	_, ended := s.ended[season]
	return ended, nil
}

func (s *Store) Count(_ context.Context, take rounds.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for _, at := range s.rounds {
		round := at.round
		if !round.Own || take.At.Before(round.StartsAt) || !take.At.Before(round.EndsAt) || take.Position < at.countedBelow {
			continue
		}
		at.countedBelow = take.Position + 1
		at.countries[take.Country]++
		if at.players[take.Account] == nil {
			at.players[take.Account] = map[rounds.Country]uint64{}
		}
		at.players[take.Account][take.Country]++
	}
	return nil
}

func (s *Store) ForgetAccount(_ context.Context, account rounds.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for _, at := range s.rounds {
		delete(at.players, account)
	}
	return nil
}

// CountedIn is what a round counted of its takes, for the contract suite.
func (s *Store) CountedIn(round rounds.Round) rounds.Counted {
	s.mu.Lock()
	defer s.mu.Unlock()

	counted := rounds.Counted{Countries: map[rounds.Country]uint64{}, Players: map[rounds.AccountID]map[rounds.Country]uint64{}}
	at, known := s.rounds[keyOf(round)]
	if !known {
		return counted
	}
	maps.Copy(counted.Countries, at.countries)
	for account, countries := range at.players {
		counted.Players[account] = maps.Clone(countries)
	}
	return counted
}
