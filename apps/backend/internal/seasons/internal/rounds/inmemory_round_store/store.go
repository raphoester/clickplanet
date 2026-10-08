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
}

func keyOf(round rounds.Round) key {
	return key{season: round.Season, endsAt: round.EndsAt.UnixNano()}
}

type kept struct {
	round   rounds.Round
	held    map[rounds.Country]uint64
	closed  bool
	results []rounds.Result
}

type Store struct {
	mu       sync.Mutex
	rounds   map[key]*kept
	failWith error
}

var _ rounds.Store = (*Store)(nil)

func New() *Store {
	return &Store{rounds: map[key]*kept{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) RecordSnapshot(_ context.Context, round rounds.Round, snapshot rounds.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	at, known := s.rounds[keyOf(round)]
	if !known {
		at = &kept{round: round, held: map[rounds.Country]uint64{}}
		s.rounds[keyOf(round)] = at
	}
	for country, tiles := range snapshot.Held {
		if tiles > 0 {
			at.held[country] += uint64(tiles)
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
	slices.SortFunc(ended, func(a, b rounds.Round) int {
		return cmp.Or(a.EndsAt.Compare(b.EndsAt), cmp.Compare(a.Season, b.Season))
	})
	return ended, nil
}

func (s *Store) Held(_ context.Context, round rounds.Round) (map[rounds.Country]uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	at, known := s.rounds[keyOf(round)]
	if !known {
		return map[rounds.Country]uint64{}, nil
	}
	return maps.Clone(at.held), nil
}

func (s *Store) Close(_ context.Context, round rounds.Round, results []rounds.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	at, known := s.rounds[keyOf(round)]
	if !known || at.closed {
		return nil
	}
	at.closed = true
	at.results = slices.Clone(results)
	return nil
}

func (s *Store) Results(round rounds.Round) []rounds.Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, known := s.rounds[keyOf(round)]
	if !known {
		return nil
	}
	return slices.Clone(at.results)
}
