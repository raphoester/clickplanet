//go:build testing

package inmemory_seen_storage

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
)

func New() *Storage {
	return &Storage{kept: map[messages.AccountID]time.Time{}}
}

type Storage struct {
	mu   sync.Mutex
	kept map[messages.AccountID]time.Time
	err  error
}

var _ seen.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) SeenUntil(_ context.Context, account messages.AccountID) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return time.Time{}, s.err
	}
	return s.kept[account], nil
}

func (s *Storage) SaveSeen(_ context.Context, account messages.AccountID, until time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	if until.After(s.kept[account]) {
		s.kept[account] = until
	}
	return nil
}

func (s *Storage) DeleteSeen(_ context.Context, account messages.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	delete(s.kept, account)
	return nil
}
