//go:build testing

package inmemory_contribution_store

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type key struct {
	season  calendar.Number
	account standings.AccountID
}

type Store struct {
	mu       sync.Mutex
	tallies  map[key]standings.Tally
	failWith error
}

var _ standings.Store = (*Store)(nil)

func New() *Store {
	return &Store{tallies: map[key]standings.Tally{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) RecordTake(_ context.Context, season calendar.Number, take standings.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	at := key{season: season, account: take.Account}
	s.tallies[at] = s.tallies[at].WithTake(take.Country)
	return nil
}

func (s *Store) Line(_ context.Context, season calendar.Number, account standings.AccountID) (standings.Line, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return standings.Line{}, s.failWith
	}
	tally, ok := s.tallies[key{season: season, account: account}]
	if !ok {
		return standings.Line{}, standings.ErrNoLine
	}
	return tally.LineOf(account), nil
}

func (s *Store) Lines(
	_ context.Context,
	season calendar.Number,
	country standings.Country,
	from standings.Cursor,
	limit int,
) ([]standings.Line, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var lines []standings.Line
	for at, tally := range s.tallies {
		line := tally.LineOf(at.account)
		if at.season == season && (country == "" || line.Country == country) && from.Before(line) {
			lines = append(lines, line)
		}
	}
	slices.SortFunc(lines, func(a, b standings.Line) int {
		if a.Above(b) {
			return -1
		}
		return 1
	})
	return lines[:min(limit, len(lines))], nil
}

func (s *Store) DeleteAccount(_ context.Context, account standings.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for at := range s.tallies {
		if at.account == account {
			delete(s.tallies, at)
		}
	}
	return nil
}
