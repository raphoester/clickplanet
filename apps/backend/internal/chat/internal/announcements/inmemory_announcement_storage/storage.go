//go:build testing

package inmemory_announcement_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []announcements.Announcement
}

var _ announcements.Storage = (*Storage)(nil)

func (s *Storage) Append(_ context.Context, announcement announcements.Announcement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slices.ContainsFunc(s.kept, func(kept announcements.Announcement) bool { return kept.ID() == announcement.ID() }) {
		return announcements.ErrKept
	}
	s.kept = append(s.kept, announcement)
	return nil
}

func (s *Storage) Kept() []announcements.Announcement {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.kept)
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(announcement announcements.Announcement) bool {
		return announcement.At().Before(cutoff)
	})
	return int64(before - len(s.kept)), nil
}
