// Package inmemory_player_storage holds every profile and every stats in memory, and writes what changed
// through its players.Persistence port every flush. A request or an event never waits on the database.
package inmemory_player_storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Config struct {
	// How often what changed is written to postgres (default 1s, and once more on shutdown).
	FlushInterval time.Duration
}

const defaultFlushInterval = time.Second

func (c Config) WithDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}
	return c
}

type Storage struct {
	persistence players.Persistence

	mu       sync.RWMutex
	profiles map[players.AccountID]players.Profile
	stats    map[players.AccountID]players.Stats
	// The accounts whose rows changed since the last flush.
	dirty *cpcolls.Set[players.AccountID]

	// One flush at a time, so a slow one cannot be overtaken by the next and write older rows last.
	flushMu sync.Mutex
}

func New(persistence players.Persistence) *Storage {
	return &Storage{
		persistence: persistence,
		profiles:    map[players.AccountID]players.Profile{},
		stats:       map[players.AccountID]players.Stats{},
		dirty:       cpcolls.NewSet[players.AccountID](),
	}
}

// Load refuses rather than start empty: an empty storage that then flushes would lose every name set since.
func (s *Storage) Load(ctx context.Context) error {
	snapshot, err := s.persistence.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the stored players: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, profile := range snapshot.Profiles {
		s.profiles[profile.Account] = profile
	}
	for _, stats := range snapshot.Stats {
		s.stats[stats.Account] = stats
	}

	return nil
}

// Profile is the account's profile, if it chose a name.
func (s *Storage) Profile(account players.AccountID) (players.Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, ok := s.profiles[account]
	return profile, ok
}

// Stats is the account's stats, if it ever took a tile.
func (s *Storage) Stats(account players.AccountID) (players.Stats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, ok := s.stats[account]
	return stats, ok
}

// Names is the name of every account given that has one.
func (s *Storage) Names(accounts []players.AccountID) map[players.AccountID]players.Name {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make(map[players.AccountID]players.Name, len(accounts))
	for _, account := range accounts {
		if profile, ok := s.profiles[account]; ok {
			names[account] = profile.Name
		}
	}
	return names
}

func (s *Storage) SaveProfile(profile players.Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.profiles[profile.Account] = profile
	s.dirty.Add(profile.Account)
}

// RecordTake counts one more tile taken by the account, under the lock, so a deletion cannot be overwritten
// by stats read before it.
func (s *Storage) RecordTake(account players.AccountID, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats, ok := s.stats[account]
	if !ok {
		stats = players.Stats{Account: account}
	}
	s.stats[account] = stats.WithTake(at)
	s.dirty.Add(account)
}

// DeleteAccount forgets the account's profile and stats.
func (s *Storage) DeleteAccount(account players.AccountID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.profiles, account)
	delete(s.stats, account)
	s.dirty.Add(account)
}

// Flush writes every account changed since the last flush, as it is now. A failed write keeps them for the next.
func (s *Storage) Flush(ctx context.Context) error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	dirty := s.dirty
	s.dirty = cpcolls.NewSet[players.AccountID]()
	changes := s.changesLocked(dirty)
	s.mu.Unlock()

	if changes.Empty() {
		return nil
	}

	if err := s.persistence.Save(ctx, changes); err != nil {
		s.mu.Lock()
		s.dirty.AddSet(dirty)
		s.mu.Unlock()
		return fmt.Errorf("failed to save the players: %w", err)
	}

	return nil
}

func (s *Storage) changesLocked(dirty *cpcolls.Set[players.AccountID]) players.Changes {
	var changes players.Changes
	dirty.ForEach(func(account players.AccountID) {
		if profile, ok := s.profiles[account]; ok {
			changes.Profiles = append(changes.Profiles, profile)
		} else {
			changes.DeletedProfiles = append(changes.DeletedProfiles, account)
		}
		if stats, ok := s.stats[account]; ok {
			changes.Stats = append(changes.Stats, stats)
		} else {
			changes.DeletedStats = append(changes.DeletedStats, account)
		}
	})
	return changes
}
