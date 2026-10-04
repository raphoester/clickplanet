package inmemory_visit_storage

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	MaxVisitsPerTag  = 10
	MaxVisits        = 10_000
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

func (s *Storage) Record(visit presence.Visit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	held, known := s.visits[visit.Account()]
	if known {
		visit = visit.Keyed(held.Key())
	} else {
		if len(s.visits) >= MaxVisits {
			return
		}
		s.makeRoomOnTag(visit.Tag())
		s.lastKey++
		visit = visit.Keyed(presence.Key(strconv.FormatUint(s.lastKey, 36)))
	}

	s.visits[visit.Account()] = visit

	if !known || !held.Fresh(s.clock.Now()) || presence.EntryOf(held) != presence.EntryOf(visit) {
		s.publish(presence.ChangeOf(presence.EntryOf(visit)))
	}
}

func (s *Storage) Move(from, to players.AccountID, author wearing.Author) {
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
		s.publish(presence.ChangeOf(presence.EntryOf(moved)))
	}
}

func (s *Storage) Rename(account players.AccountID, username players.Name) {
	s.mu.Lock()
	defer s.mu.Unlock()

	visit, held := s.visits[account]
	if !held {
		return
	}

	renamed := visit.For(account, visit.Author().Renamed(username))
	s.visits[account] = renamed

	if presence.EntryOf(visit) != presence.EntryOf(renamed) {
		s.publish(presence.ChangeOf(presence.EntryOf(renamed)))
	}
}

func (s *Storage) Wear(account players.AccountID, worn titles.Standing) {
	s.mu.Lock()
	defer s.mu.Unlock()

	visit, held := s.visits[account]
	if !held {
		return
	}

	dressed := visit.For(account, visit.Author().Wearing(worn))
	s.visits[account] = dressed

	if presence.EntryOf(visit) != presence.EntryOf(dressed) {
		s.publish(presence.ChangeOf(presence.EntryOf(dressed)))
	}
}

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
		if held.Tag() != tag {
			continue
		}
		count++
		if count == 1 || held.At().Before(oldest.At()) {
			oldest = held
		}
	}

	if count >= MaxVisitsPerTag {
		s.drop(oldest.Account())
	}
}

func (s *Storage) drop(account players.AccountID) {
	visit, held := s.visits[account]
	if !held {
		return
	}

	delete(s.visits, account)
	s.publish(presence.DepartureOf(presence.EntryOf(visit)))
}

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

// publish must run under s.mu, so each subscriber reads changes in order.
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
