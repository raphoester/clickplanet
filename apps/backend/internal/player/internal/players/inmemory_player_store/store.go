//go:build testing

package inmemory_player_store

import (
	"bytes"
	"context"
	"slices"
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

func (s *Store) SaveProfile(_ context.Context, profile players.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for account, held := range s.profiles {
		if account != profile.Account() && held.Name().Folded() == profile.Name().Folded() {
			return players.ErrNameTaken
		}
	}
	held := s.profiles[profile.Account()]
	s.profiles[profile.Account()] = players.ProfileOf(profile.Account(), profile.Name(), profile.UpdatedAt(), held.Admin(), held.Color())
	return nil
}

func (s *Store) CreateProfile(_ context.Context, profile players.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, ok := s.profiles[profile.Account()]; ok {
		return players.ErrProfileExists
	}
	for _, held := range s.profiles {
		if held.Name().Folded() == profile.Name().Folded() {
			return players.ErrNameTaken
		}
	}
	s.profiles[profile.Account()] = profile
	return nil
}

func (s *Store) SaveColor(_ context.Context, account players.AccountID, color players.Color) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	profile, ok := s.profiles[account]
	if !ok {
		return players.ErrNoProfile
	}
	s.profiles[account] = players.ProfileOf(profile.Account(), profile.Name(), profile.UpdatedAt(), profile.Admin(), color)
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

func (s *Store) MakeAdmin(account players.AccountID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.profiles[account]
	s.profiles[account] = players.ProfileOf(profile.Account(), profile.Name(), profile.UpdatedAt(), true, profile.Color())
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
		stats = players.NewStats(account)
	}
	s.stats[account] = stats.WithTake(at)
	return nil
}

func (s *Store) SaveTakes(counted players.Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	account := counted.Account()
	kept, ok := s.stats[account]
	if !ok {
		kept = players.NewStats(account)
	}
	s.stats[account] = players.StatsOf(account, counted.TilesTaken(), counted.Streak(), counted.StreakBest(), kept.MessagesSent())
}

func (s *Store) RecordMessage(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	stats, ok := s.stats[account]
	if !ok {
		stats = players.NewStats(account)
	}
	s.stats[account] = stats.WithMessage()
	return nil
}

func (s *Store) StatsAfter(_ context.Context, after players.AccountID, limit int) ([]players.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var page []players.Stats
	for account, stats := range s.stats {
		if bytes.Compare(account[:], after[:]) > 0 {
			page = append(page, stats)
		}
	}
	slices.SortFunc(page, func(a, b players.Stats) int { return bytes.Compare(accountBytes(a), accountBytes(b)) })
	return page[:min(limit, len(page))], nil
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
			names[account] = profile.Name()
		}
	}
	return names, nil
}

func accountBytes(stats players.Stats) []byte {
	account := stats.Account()
	return account[:]
}
