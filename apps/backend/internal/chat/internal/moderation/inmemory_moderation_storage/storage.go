//go:build testing

package inmemory_moderation_storage

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

func New() *Storage {
	return &Storage{}
}

type Storage struct {
	mu   sync.Mutex
	kept []moderation.Snapshot
	err  error
}

var _ moderation.Storage = (*Storage)(nil)

// FailWith makes every call answer err.
func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *Storage) Record(_ context.Context, action moderation.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	recorded := action.TakeSnapshot()
	if recorded.Kind == moderation.Removed && slices.ContainsFunc(s.kept, func(kept moderation.Snapshot) bool {
		return kept.Kind == moderation.Removed && kept.Message == recorded.Message
	}) {
		return nil
	}
	s.kept = append(s.kept, recorded)
	return nil
}

func (s *Storage) Kept() []moderation.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.kept)
}

func (s *Storage) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.kept)
	s.kept = slices.DeleteFunc(s.kept, func(kept moderation.Snapshot) bool { return kept.At.Before(cutoff) })
	return int64(before - len(s.kept)), nil
}

func (s *Storage) Forget(_ context.Context, account messages.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.kept = slices.DeleteFunc(s.kept, func(kept moderation.Snapshot) bool { return kept.Account == account })
	for i, kept := range s.kept {
		if kept.By.Account() == account {
			s.kept[i].By = moderation.OperatorOf(messages.NoAccount, kept.By.Client())
		}
	}
	return nil
}
