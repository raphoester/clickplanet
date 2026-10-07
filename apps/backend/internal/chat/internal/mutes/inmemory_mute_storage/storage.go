//go:build testing

package inmemory_mute_storage

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []mutes.Mute
}

var _ mutes.Storage = (*Storage)(nil)

var errDuplicateID = errors.New("a mute with this id is already kept")

func (s *Storage) Save(_ context.Context, mute mutes.Mute) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slices.ContainsFunc(s.kept, func(kept mutes.Mute) bool { return kept.ID() == mute.ID() }) {
		return errDuplicateID
	}
	s.kept = append(s.kept, mute)
	return nil
}

func (s *Storage) Kept() []mutes.Mute {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.kept)
}

func (s *Storage) Mute(_ context.Context, caller mutes.Caller, at time.Time) (mutes.Mute, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var found *mutes.Mute
	for i, mute := range s.kept {
		if mute.ApplicableTo(caller, at) && (found == nil || mute.Until().After(found.Until())) {
			found = &s.kept[i]
		}
	}
	if found == nil {
		return mutes.Mute{}, mutes.ErrNotMuted
	}
	return *found, nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(mute mutes.Mute) bool { return mute.Until().Before(cutoff) })
	return int64(before - len(s.kept)), nil
}
