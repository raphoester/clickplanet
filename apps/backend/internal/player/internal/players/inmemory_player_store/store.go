//go:build testing

// Package inmemory_player_store is players.Store in maps, for the tests above the port. It can fail on demand.
package inmemory_player_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store struct {
	mu       sync.Mutex
	profiles map[players.AccountID]players.Profile
	codes    map[players.AccountID]players.GuestCode
	stats    map[players.AccountID]players.Stats
	failWith error
}

var _ players.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		profiles: map[players.AccountID]players.Profile{},
		codes:    map[players.AccountID]players.GuestCode{},
		stats:    map[players.AccountID]players.Stats{},
	}
}

// FailWith makes every later call answer err.
func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Profile(_ context.Context, account players.AccountID) (players.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return players.Profile{}, s.failWith
	}
	profile, ok := s.profiles[account]
	if !ok {
		return players.Profile{}, players.ErrNoProfile
	}
	return profile, nil
}

func (s *Store) ProfileNamed(_ context.Context, name players.Name) (players.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return players.Profile{}, s.failWith
	}
	for _, profile := range s.profiles {
		if profile.Name.Folded() == name.Folded() {
			return profile, nil
		}
	}
	return players.Profile{}, players.ErrNoProfile
}

func (s *Store) SaveProfile(_ context.Context, profile players.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for account, held := range s.profiles {
		if account != profile.Account && held.Name.Folded() == profile.Name.Folded() {
			return players.ErrNameTaken
		}
	}
	profile.Admin = s.profiles[profile.Account].Admin
	s.profiles[profile.Account] = profile
	return nil
}

func (s *Store) GuestCode(_ context.Context, account players.AccountID) (players.GuestCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return "", s.failWith
	}
	code, ok := s.codes[account]
	if !ok {
		return "", players.ErrNoGuestCode
	}
	return code, nil
}

func (s *Store) SaveGuestCode(_ context.Context, account players.AccountID, code players.GuestCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, ok := s.codes[account]; ok {
		return nil
	}
	for _, held := range s.codes {
		if held == code {
			return players.ErrGuestCodeTaken
		}
	}
	s.codes[account] = code
	return nil
}

// MakeAdmin is what an operator does in the database: the game has no way to.
func (s *Store) MakeAdmin(account players.AccountID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.profiles[account]
	profile.Admin = true
	s.profiles[account] = profile
}

func (s *Store) Stats(_ context.Context, account players.AccountID) (players.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return players.Stats{}, s.failWith
	}
	stats, ok := s.stats[account]
	if !ok {
		return players.Stats{}, players.ErrNoStats
	}
	return stats, nil
}

func (s *Store) RecordTake(_ context.Context, account players.AccountID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	stats, ok := s.stats[account]
	if !ok {
		stats = players.Stats{Account: account}
	}
	s.stats[account] = stats.WithTake(at)
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.profiles, account)
	delete(s.codes, account)
	delete(s.stats, account)
	return nil
}

func (s *Store) Names(_ context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	names := make(map[players.AccountID]players.Name, len(accounts))
	for _, account := range accounts {
		if profile, ok := s.profiles[account]; ok {
			names[account] = profile.Name
		}
	}
	return names, nil
}
