package inmemory_grant_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

// Store keeps each grant for keep, in memory: an offer's grant, named by a token that lives seconds, is paid in the
// claim that took it, so nothing of a box or a quiz is written to postgres. A grant that must outlive a restart (the
// sign-in refill's, a GrantCharges command's) is kept in charge_grants (postgres_grant_store).
type Store struct {
	keep time.Duration

	mu       sync.Mutex
	recorded map[charges.GrantID]time.Time
	swept    time.Time
}

func New(keep time.Duration) *Store {
	return &Store{keep: keep, recorded: make(map[charges.GrantID]time.Time)}
}

// Record is not undone by the payment's rollback: a payment that failed is lost, as a box whose grant failed is.
func (s *Store) Record(_ context.Context, grant charges.GrantID, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.forget(at)
	if _, kept := s.recorded[grant]; kept {
		return false, nil
	}
	s.recorded[grant] = at

	return true, nil
}

// forget walks the grants at most once per keep, so each is kept for keep at least and twice that at most.
func (s *Store) forget(now time.Time) {
	if now.Sub(s.swept) < s.keep {
		return
	}

	for grant, at := range s.recorded {
		if now.Sub(at) >= s.keep {
			delete(s.recorded, grant)
		}
	}
	s.swept = now
}
