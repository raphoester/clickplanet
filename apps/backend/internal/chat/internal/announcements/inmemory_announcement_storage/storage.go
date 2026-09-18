//go:build testing

// Package inmemory_announcement_storage keeps announcements in a slice, for tests that need an
// announcements.Storage but not postgres.
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

	s.kept = append(s.kept, announcement)
	return nil
}

func (s *Storage) Recent(_ context.Context, since time.Time, limit int) ([]announcements.Announcement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var recent []announcements.Announcement
	for i := len(s.kept) - 1; i >= 0 && len(recent) < limit; i-- {
		if !s.kept[i].At.Before(since) {
			recent = append(recent, s.kept[i])
		}
	}
	slices.Reverse(recent)
	return recent, nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(announcement announcements.Announcement) bool {
		return announcement.At.Before(cutoff)
	})
	return int64(before - len(s.kept)), nil
}
