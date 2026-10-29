//go:build testing

package inmemory_day_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

var _ days.Store = (*Store)(nil)

func New(told ...days.Day) *Store {
	store := &Store{days: map[days.DayID]days.Day{}}
	for _, day := range told {
		store.days[day.Snapshot().ID] = day
	}
	return store
}

type Store struct {
	mu   sync.Mutex
	days map[days.DayID]days.Day
}

func (s *Store) DayAt(_ context.Context, at time.Time) (days.Day, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, day := range s.days {
		if day.Runs(at) {
			return day, nil
		}
	}
	return days.Day{}, days.ErrNoDay
}

func (s *Store) DayBefore(_ context.Context, day days.Day) (days.Day, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, told := range s.days {
		if day.Follows(told) {
			return told, nil
		}
	}
	return days.Day{}, days.ErrNoDay
}

func (s *Store) KeepDay(_ context.Context, day days.Day) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.days[day.Snapshot().ID] = day
	return nil
}
