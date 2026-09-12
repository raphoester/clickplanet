package bonus

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Spreads remembers who holds a running spread bonus. It is the spread's
// counterpart to the limiter's boost: the claim starts one, and the click chain
// reads it on every click.
type Spreads struct {
	clock cptime.Clock

	mu    sync.RWMutex
	until map[string]time.Time
}

func NewSpreads(clock cptime.Clock) *Spreads {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Spreads{clock: clock, until: make(map[string]time.Time)}
}

// Grant starts a spread for scope that runs until the given time.
//
// It also forgets every spread that has run out. That keeps the map as small as
// the number of bonuses running, with no sweep of its own: a grant is rare, and
// the map holds only the callers who caught one in the last minute.
func (s *Spreads) Grant(scope string, until time.Time) {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for other, end := range s.until {
		if !now.Before(end) {
			delete(s.until, other)
		}
	}

	s.until[scope] = until
}

// Spreading reports whether scope's clicks spread right now.
func (s *Spreads) Spreading(scope string) bool {
	now := s.clock.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()

	end, ok := s.until[scope]

	return ok && now.Before(end)
}
