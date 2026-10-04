//go:build testing

package inmemory_take_store

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
)

type Stats interface {
	Stats(ctx context.Context, account players.AccountID) (players.Stats, error)
	StatsAfter(ctx context.Context, after players.AccountID, limit int) ([]players.Stats, error)
	SaveTakes(counted players.Stats)
}

type Store struct {
	stats Stats

	mu       sync.Mutex
	started  bool
	start    takes.Position
	position takes.Position
	baseline map[players.AccountID]players.Stats
	failWith error
}

var _ takes.Store = (*Store)(nil)

func New(stats Stats) *Store {
	return &Store{stats: stats, baseline: map[players.AccountID]players.Stats{}}
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

func (s *Store) Position(context.Context) (takes.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	if !s.started {
		return 0, takes.ErrNotStarted
	}
	return s.position, nil
}

func (s *Store) Begin(ctx context.Context, start takes.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if s.started {
		return takes.ErrStarted
	}

	all, err := s.stats.StatsAfter(ctx, players.AccountID{}, math.MaxInt)
	if err != nil {
		return err //nolint:wrapcheck // a fake.
	}
	for _, stats := range all {
		if stats.TilesTaken() > 0 {
			s.baseline[stats.Account()] = stats
		}
	}

	s.started, s.start, s.position = true, start, start
	return nil
}

func (s *Store) Count(ctx context.Context, batch takes.Batch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if !s.started {
		return takes.ErrNotStarted
	}
	if s.position != batch.From() {
		return takes.ErrMoved
	}

	current := map[players.AccountID]players.Stats{}
	for _, account := range batch.Accounts() {
		stats, err := s.stats.Stats(ctx, account)
		if errors.Is(err, players.ErrNoStats) {
			continue
		}
		if err != nil {
			return err //nolint:wrapcheck // a fake.
		}
		current[account] = stats
	}

	for _, counted := range batch.Tallied(current) {
		s.stats.SaveTakes(counted)
	}
	s.position = batch.Next()
	return nil
}

func (s *Store) Rewind(ctx context.Context) (takes.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return 0, s.failWith
	}
	if !s.started {
		return 0, takes.ErrNotStarted
	}

	all, err := s.stats.StatsAfter(ctx, players.AccountID{}, math.MaxInt)
	if err != nil {
		return 0, err //nolint:wrapcheck // a fake.
	}
	for _, stats := range all {
		baseline, ok := s.baseline[stats.Account()]
		if !ok {
			baseline = players.NewStats(stats.Account())
		}
		s.stats.SaveTakes(baseline)
	}

	s.position = s.start
	return s.start, nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.baseline, account)
	return nil
}
