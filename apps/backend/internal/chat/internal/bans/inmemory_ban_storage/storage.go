// Package inmemory_ban_storage holds the chat's bans in memory and writes every change through.
package inmemory_ban_storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

const writeTimeout = 5 * time.Second

func New(persistence Persistence, logger *slog.Logger) *Storage {
	return &Storage{
		persistence: persistence,
		logger:      logger,
		banned:      make(map[string]bans.Ban),
	}
}

// Storage is read on every message sent and on every history served, so the set
// lives here and postgres is only where it is kept.
type Storage struct {
	persistence Persistence
	logger      *slog.Logger

	writeMu sync.Mutex

	mu     sync.RWMutex
	banned map[string]bans.Ban
}

func (s *Storage) Banned(tag string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, banned := s.banned[tag]
	return banned
}

// Ban records the ban before it is enforced, and keeps the time the member was
// first silenced when one is already running.
func (s *Storage) Ban(ctx context.Context, ban bans.Ban) (bans.Ban, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if running := s.find(ban.AuthorTag); running != nil {
		ban.BannedAt = running.BannedAt
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := s.persistence.Upsert(ctx, ban); err != nil {
		return bans.Ban{}, fmt.Errorf("failed to record the chat ban: %w", err)
	}

	s.mu.Lock()
	s.banned[ban.AuthorTag] = ban
	s.mu.Unlock()

	return ban, nil
}

func (s *Storage) Unban(ctx context.Context, tag string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.find(tag) == nil {
		return fmt.Errorf("%w: %s", bans.ErrNotBanned, tag)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := s.persistence.Delete(ctx, tag); err != nil {
		return fmt.Errorf("failed to lift the chat ban: %w", err)
	}

	s.mu.Lock()
	delete(s.banned, tag)
	s.mu.Unlock()

	return nil
}

// All is every running ban, newest first.
func (s *Storage) All() []bans.Ban {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]bans.Ban, 0, len(s.banned))
	for _, ban := range s.banned {
		all = append(all, ban)
	}

	slices.SortFunc(all, func(a, b bans.Ban) int {
		if !a.BannedAt.Equal(b.BannedAt) {
			return b.BannedAt.Compare(a.BannedAt)
		}
		return strings.Compare(a.AuthorTag, b.AuthorTag)
	})

	return all
}

// find is nil when nothing is running, rather than a second return value.
func (s *Storage) find(tag string) *bans.Ban {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ban, banned := s.banned[tag]
	if !banned {
		return nil
	}
	return &ban
}
