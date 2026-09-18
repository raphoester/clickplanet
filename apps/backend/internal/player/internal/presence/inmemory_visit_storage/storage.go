// Package inmemory_visit_storage keeps the last visit of every player in memory, and publishes each change.
package inmemory_visit_storage

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	// MaxVisitsPerTag bounds the accounts one address holds on the roster: a campus fits, and a script minting
	// accounts on one address pushes out only its own.
	MaxVisitsPerTag = 10
	// MaxVisits bounds the whole roster. A new account past it is not recorded; one already on it still is.
	MaxVisits = 10_000
	// SubscriberBuffer: a subscriber further behind is closed, not skipped, so it reconnects to a whole roster.
	SubscriberBuffer = 256

	pruneInterval = 5 * time.Second
)

type Storage struct {
	clock cptime.Clock

	mu          sync.Mutex
	visits      map[players.AccountID]presence.Visit
	lastKey     uint64
	subscribers *cpcolls.Set[chan presence.Change]
}

func New(clock cptime.Clock) *Storage {
	return &Storage{
		clock:       clock,
		visits:      map[players.AccountID]presence.Visit{},
		subscribers: cpcolls.NewSet[chan presence.Change](),
	}
}

// Record keeps the visit over the account's last one, and its key. A new account gets a new key, and pushes
// out the oldest visit of its tag once the tag holds MaxVisitsPerTag.
func (s *Storage) Record(visit presence.Visit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	held, known := s.visits[visit.Account]
	if known {
		visit.Key = held.Key
	} else {
		if len(s.visits) >= MaxVisits {
			return
		}
		s.makeRoomOnTag(visit.Tag)
		s.lastKey++
		visit.Key = presence.Key(strconv.FormatUint(s.lastKey, 36))
	}

	s.visits[visit.Account] = visit

	// A stale visit was on no roster a subscriber read.
	if !known || !held.Fresh(s.clock.Now()) || presence.EntryOf(held) != presence.EntryOf(visit) {
		s.publish(presence.Change{Entry: presence.EntryOf(visit)})
	}
}

// Move carries a signed-in browser's visit, and its key, to its account's name, over any visit the account held.
func (s *Storage) Move(from, to players.AccountID, author players.Author) {
	s.mu.Lock()
	defer s.mu.Unlock()

	visit, held := s.visits[from]
	if !held {
		return
	}

	if from != to {
		s.drop(to)
	}

	delete(s.visits, from)
	moved := visit.For(to, author)
	s.visits[to] = moved

	if presence.EntryOf(visit) != presence.EntryOf(moved) {
		s.publish(presence.Change{Entry: presence.EntryOf(moved)})
	}
}

// Rename shows the account under its new username. An admin stays one.
func (s *Storage) Rename(account players.AccountID, username players.Name) {
	s.mu.Lock()
	defer s.mu.Unlock()

	visit, held := s.visits[account]
	if !held {
		return
	}

	renamed := visit.For(account, players.Author{Name: players.DisplayNameOf(username, ""), Admin: visit.Author.Admin})
	s.visits[account] = renamed

	if presence.EntryOf(visit) != presence.EntryOf(renamed) {
		s.publish(presence.Change{Entry: presence.EntryOf(renamed)})
	}
}

// Forget takes the account off the roster at once, rather than when its last visit goes stale.
func (s *Storage) Forget(account players.AccountID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.drop(account)
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
		s.drop(oldest.Account)
	}
}

func (s *Storage) drop(account players.AccountID) {
	visit, held := s.visits[account]
	if !held {
		return
	}

	delete(s.visits, account)
	s.publish(presence.Change{Entry: presence.EntryOf(visit), Left: true})
}

// Visits is every visit held, stale ones included until the next prune.
func (s *Storage) Visits() []presence.Visit {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.held()
}

func (s *Storage) held() []presence.Visit {
	visits := make([]presence.Visit, 0, len(s.visits))
	for _, visit := range s.visits {
		visits = append(visits, visit)
	}
	return visits
}

// Subscribe answers the roster and every change after it. The channel closes with ctx, or when it falls behind.
func (s *Storage) Subscribe(ctx context.Context) ([]presence.Entry, <-chan presence.Change) {
	changes := make(chan presence.Change, SubscriberBuffer)

	s.mu.Lock()
	roster := presence.RosterOf(s.held(), s.clock.Now())
	s.subscribers.Add(changes)
	s.mu.Unlock()

	go func() {
		<-ctx.Done()

		s.mu.Lock()
		defer s.mu.Unlock()

		s.unsubscribe(changes)
	}()

	return roster, changes
}

// publish runs under the lock, so each subscriber reads changes in order.
func (s *Storage) publish(change presence.Change) {
	var behind []chan presence.Change
	s.subscribers.ForEach(func(changes chan presence.Change) {
		select {
		case changes <- change:
		default:
			behind = append(behind, changes)
		}
	})
	for _, changes := range behind {
		s.unsubscribe(changes)
	}
}

func (s *Storage) unsubscribe(changes chan presence.Change) {
	if !s.subscribers.Contains(changes) {
		return
	}

	s.subscribers.Delete(changes)
	close(changes)
}

// Prune forgets every visit that is no longer fresh.
func (s *Storage) Prune() {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for account, visit := range s.visits {
		if !visit.Fresh(now) {
			s.drop(account)
		}
	}
}

func (s *Storage) Name() string { return "player-presence-prune" }

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
