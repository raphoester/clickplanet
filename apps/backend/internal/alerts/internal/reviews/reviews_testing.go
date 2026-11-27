//go:build testing

package reviews

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ Store = (*InMemoryStore)(nil)

type InMemoryStore struct {
	mu   sync.Mutex
	told map[string]time.Time
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{told: map[string]time.Time{}}
}

func (s *InMemoryStore) Told(_ context.Context, submission SubmissionID, cause Cause) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.told[string(submission)+"/"+string(cause)]
	return ok, nil
}

// Kept once the fake transaction commits, as postgres keeps it.
func (s *InMemoryStore) SaveTold(ctx context.Context, submission SubmissionID, cause Cause, at time.Time) error {
	cpatomic.OnCommit(ctx, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.told[string(submission)+"/"+string(cause)] = at
	})
	return nil
}
