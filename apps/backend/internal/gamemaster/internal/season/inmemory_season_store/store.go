//go:build testing

package inmemory_season_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Store struct {
	mu       sync.Mutex
	seasons  map[season.Number]season.Snapshot
	failWith error
}

var _ season.Store = (*Store)(nil)

func New() *Store {
	return &Store{seasons: map[season.Number]season.Snapshot{}}
}

func (s *Store) Calendar(context.Context) (season.Calendar, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return season.Calendar{}, s.failWith
	}
	seasons := make([]season.Season, 0, len(s.seasons))
	for _, kept := range s.seasons {
		seasons = append(seasons, kept.Restore())
	}
	return season.NewCalendar(seasons...), nil
}

func (s *Store) Save(_ context.Context, kept season.Season) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	snapshot := kept.TakeSnapshot()
	s.seasons[snapshot.Number] = snapshot
	return nil
}

func (s *Store) Delete(_ context.Context, number season.Number) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.seasons, number)
	return nil
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}
