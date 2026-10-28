//go:build testing

package inmemory_player_store

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store struct {
	mu              sync.Mutex
	profiles        map[players.AccountID]players.Profile
	hiddenFromClips map[players.AccountID]bool
	codes           map[players.AccountID]players.GuestCode
	stats           map[players.AccountID]players.Stats
	failWith        error
}

var _ players.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		profiles:        map[players.AccountID]players.Profile{},
		hiddenFromClips: map[players.AccountID]bool{},
		codes:           map[players.AccountID]players.GuestCode{},
		stats:           map[players.AccountID]players.Stats{},
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
	saved := profile.TakeSnapshot()
	for account, held := range s.profiles {
		if account != saved.Account && held.TakeSnapshot().Name.Folded() == saved.Name.Folded() {
			return players.ErrNameTaken
		}
	}
	held := s.profiles[saved.Account].TakeSnapshot()
	saved.Admin, saved.Color = held.Admin, held.Color
	s.profiles[saved.Account] = saved.Restore()
	return nil
}

func (s *Store) CreateProfile(_ context.Context, profile players.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	created := profile.TakeSnapshot()
	if _, ok := s.profiles[created.Account]; ok {
		return players.ErrProfileExists
	}
	for _, held := range s.profiles {
		if held.TakeSnapshot().Name.Folded() == created.Name.Folded() {
			return players.ErrNameTaken
		}
	}
	s.profiles[created.Account] = profile
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
	colored := profile.TakeSnapshot()
	colored.Color = color
	s.profiles[account] = colored.Restore()
	return nil
}

func (s *Store) SaveShownInClips(_ context.Context, account players.AccountID, shown bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	if _, ok := s.profiles[account]; !ok {
		return players.ErrNoProfile
	}
	s.hiddenFromClips[account] = !shown
	return nil
}

func (s *Store) ShownInClips(account players.AccountID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, named := s.profiles[account]
	return named && !s.hiddenFromClips[account]
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

	admin := s.profiles[account].TakeSnapshot()
	admin.Admin = true
	s.profiles[account] = admin.Restore()
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

func (s *Store) Record(_ context.Context, account players.AccountID, change func(players.Stats) players.Stats) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	stats, ok := s.stats[account]
	if !ok {
		stats = players.NewStats(account)
	}
	s.stats[account] = change(stats)
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
	delete(s.hiddenFromClips, account)
	delete(s.codes, account)
	delete(s.stats, account)
	return nil
}

func accountBytes(stats players.Stats) []byte {
	account := stats.Account()
	return account[:]
}
