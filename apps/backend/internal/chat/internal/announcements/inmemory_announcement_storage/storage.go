//go:build testing

// Package inmemory_announcement_storage keeps announcements in a slice, for tests that need an
// announcements.Storage but not postgres.
package inmemory_announcement_storage

import (
	"bytes"
	"cmp"
	"context"
	"errors"
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

var errDuplicateID = errors.New("an announcement with this id is already kept")

func (s *Storage) Append(_ context.Context, announcement announcements.Announcement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slices.ContainsFunc(s.kept, func(kept announcements.Announcement) bool { return kept.ID == announcement.ID }) {
		return errDuplicateID
	}
	s.kept = append(s.kept, announcement)
	return nil
}

func (s *Storage) Recent(_ context.Context, since time.Time, limit int) ([]announcements.Announcement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// By time, then by id, as postgres orders them.
	recent := slices.DeleteFunc(slices.Clone(s.kept), func(announcement announcements.Announcement) bool {
		return announcement.At.Before(since)
	})
	slices.SortStableFunc(recent, func(a, b announcements.Announcement) int {
		return cmp.Or(a.At.Compare(b.At), bytes.Compare(a.ID[:], b.ID[:]))
	})
	if len(recent) > limit {
		recent = recent[len(recent)-limit:]
	}
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
