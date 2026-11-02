//go:build testing

package inmemory_flag_day_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

type key struct {
	day  string
	flag string
}

type Store struct {
	mu       sync.Mutex
	days     map[key]flagmissions.DaySnapshot
	failWith error
}

var _ flagmissions.Store = (*Store)(nil)

func New() *Store {
	return &Store{days: map[key]flagmissions.DaySnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Day(_ context.Context, day days.Day, flag string) (flagmissions.Day, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read(key{day: day.String(), flag: flag})
}

func (s *Store) read(at key) (flagmissions.Day, error) {
	if s.failWith != nil {
		return flagmissions.Day{}, s.failWith
	}
	kept, ok := s.days[at]
	if !ok {
		return flagmissions.Day{}, flagmissions.ErrNoDay
	}
	return flagmissions.DayOf(kept) //nolint:wrapcheck // the domain named it.
}

func (s *Store) KeepDay(_ context.Context, day days.Day, flag string, drawn flagmissions.Day) (flagmissions.Day, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	at := key{day: day.String(), flag: flag}
	if s.failWith != nil {
		return s.read(at)
	}
	kept, ok := s.days[at]
	switch fresh := drawn.Snapshot(); {
	case !ok:
		s.days[at] = fresh
	case kept.Mission == nil && fresh.Mission != nil:
		kept.Mission, kept.Progress, kept.Done = fresh.Mission, fresh.Progress, fresh.Done
		s.days[at] = kept
	}
	return s.read(at)
}

func (s *Store) ChangeDay(
	_ context.Context, day days.Day, flag string, change func(flagmissions.Day) flagmissions.Day,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	at := key{day: day.String(), flag: flag}
	current, err := s.read(at)
	if err != nil {
		return err
	}
	s.days[at] = change(current).Snapshot()
	return nil
}
