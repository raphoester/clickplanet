package inmemory_ban_storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

type Persistence interface {
	All(ctx context.Context) ([]bans.Ban, error)
	Upsert(ctx context.Context, ban bans.Ban) error
	Delete(ctx context.Context, tag string) error
}

// Load fills the set at boot. A failed load refuses it: an empty set serves every
// banned member's history back and takes their next message.
func (s *Storage) Load(ctx context.Context) error {
	all, err := s.persistence.All(ctx)
	if err != nil {
		return fmt.Errorf("failed to load the chat bans: %w", err)
	}

	s.mu.Lock()
	clear(s.banned)
	for _, ban := range all {
		s.banned[ban.AuthorTag] = ban
	}
	s.mu.Unlock()

	s.logger.Info("loaded the chat bans", slog.Int("bans", len(all)))

	return nil
}
