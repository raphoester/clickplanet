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
	return &Storage{kept: map[mark]time.Time{}}
}

type mark struct {
	account messages.AccountID
	room    messages.Room
}

type Storage struct {
	mu   sync.Mutex
	kept map[mark]time.Time
	err  error
}

var _ seen.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Kept(account messages.AccountID, room messages.Room) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.kept[mark{account: account, room: room}]
}

func (s *Storage) SaveSeen(_ context.Context, account messages.AccountID, room messages.Room, until time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	at := mark{account: account, room: room}
	if until.After(s.kept[at]) {
		s.kept[at] = until
	}
	return nil
}

func (s *Storage) DeleteSeen(_ context.Context, account messages.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for at := range s.kept {
		if at.account == account {
			delete(s.kept, at)
		}
	}
	return nil
}
