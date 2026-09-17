// Package inmemory_visit_storage keeps the last visit of every player in memory, and prunes the stale ones.
package inmemory_visit_storage

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	// MaxVisitsPerTag bounds the accounts one address holds on the roster: a campus fits, and a script minting
	// accounts on one address pushes out only its own.
	MaxVisitsPerTag = 10
	// MaxVisits bounds the whole roster. A new account past it is not recorded; one already on it still is.
	MaxVisits = 10_000

	pruneInterval = time.Minute
)

type Storage struct {
	clock cptime.Clock

	mu     sync.Mutex
	visits map[players.AccountID]presence.Visit
}

func New(clock cptime.Clock) *Storage {
	return &Storage{clock: clock, visits: map[players.AccountID]presence.Visit{}}
}

// Record keeps the visit over the account's last one. An account new to the roster pushes out the oldest
// visit of its tag once the tag holds MaxVisitsPerTag.
func (s *Storage) Record(visit presence.Visit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := s.visits[visit.Account]; !known {
		if len(s.visits) >= MaxVisits {
			return
		}
		s.makeRoomOnTag(visit.Tag)
	}

	s.visits[visit.Account] = visit
}

func (s *Storage) makeRoomOnTag(tag players.Tag) {
	var (
		count  int
		oldest presence.Visit
	)
	for _, held := range s.visits {
		if held.Tag != tag {
			continue
		}
		count++
		if count == 1 || held.At.Before(oldest.At) {
			oldest = held
		}
	}

	if count >= MaxVisitsPerTag {
		delete(s.visits, oldest.Account)
	}
}

// Visits is every visit held, stale ones included until the next prune.
func (s *Storage) Visits() []presence.Visit {
	s.mu.Lock()
	defer s.mu.Unlock()

	visits := make([]presence.Visit, 0, len(s.visits))
	for _, visit := range s.visits {
		visits = append(visits, visit)
	}
	return visits
}

// Prune forgets every visit that is no longer fresh.
func (s *Storage) Prune() {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for account, visit := range s.visits {
		if !visit.Fresh(now) {
			delete(s.visits, account)
		}
	}
}

func (s *Storage) Name() string { return "player-presence-prune" }

// Run prunes every minute until the context ends.
func (s *Storage) Run(ctx context.Context) {
	ticker := time.NewTicker(pruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Prune()
		case <-ctx.Done():
			return
		}
	}
}
