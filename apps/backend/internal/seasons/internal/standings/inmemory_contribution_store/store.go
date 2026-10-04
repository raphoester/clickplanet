//go:build testing

package inmemory_contribution_store

import (
	"context"
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
	started  bool
	position standings.Position
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

func (s *Store) Heal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = nil
}

func (s *Store) Position(context.Context) (standings.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	if !s.started {
		return 0, standings.ErrNotStarted
	}
	return s.position, nil
}

func (s *Store) Begin(_ context.Context, start standings.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if s.started {
		return standings.ErrStarted
	}
	s.started, s.position = true, start
	return nil
}

func (s *Store) Count(_ context.Context, batch standings.Batch, seasons calendar.Calendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if !s.started {
		return standings.ErrNotStarted
	}
	if s.position != batch.From() {
		return standings.ErrMoved
	}
	for _, take := range batch.Takes() {
		season, ok := take.Season(seasons)
		if !ok {
			continue
		}
		at := key{season: season, account: take.Account}
		s.tallies[at] = s.tallies[at].WithTake(take.Country)
	}
	s.position = batch.Next()
	return nil
}

func (s *Store) Rewind(context.Context) (standings.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	if !s.started {
		return 0, standings.ErrNotStarted
	}
	s.tallies = map[key]standings.Tally{}
	s.position = 0
	return 0, nil
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

func (s *Store) Tally(season calendar.Number, account standings.AccountID) standings.Tally {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tallies[key{season: season, account: account}]
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
