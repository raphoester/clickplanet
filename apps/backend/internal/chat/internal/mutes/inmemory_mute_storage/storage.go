//go:build testing

package inmemory_mute_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
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

func (s *Storage) Save(_ context.Context, mute mutes.Mute) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if i := slices.IndexFunc(s.kept, func(kept mutes.Mute) bool { return kept.ID() == mute.ID() }); i >= 0 {
		s.kept[i] = mute
		return nil
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

func (s *Storage) MutesOf(_ context.Context, account messages.AccountID) ([]mutes.Mute, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var found []mutes.Mute
	for _, mute := range s.kept {
		if mute.Caller().Account() == account {
			found = append(found, mute)
		}
	}
	return found, nil
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(mute mutes.Mute) bool { return mute.Until().Before(cutoff) })
	return int64(before - len(s.kept)), nil
}
