//go:build testing

package inmemory_raid_storage

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept *raids.RaidMode
	err  error
}

var _ raids.Storage = (*Storage)(nil)

func (s *Storage) Save(_ context.Context, mode raids.RaidMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	s.kept = &mode
	return nil
}

func (s *Storage) Current(context.Context) (raids.RaidMode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.err != nil:
		return raids.RaidMode{}, s.err
	case s.kept == nil:
		return raids.RaidMode{}, raids.ErrNoRaidMode
	default:
		return *s.kept, nil
	}
}

// Every call fails with err from now on.
func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}
