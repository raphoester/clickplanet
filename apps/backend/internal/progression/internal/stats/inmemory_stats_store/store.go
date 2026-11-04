//go:build testing

package inmemory_stats_store

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type Store struct {
	mu       sync.Mutex
	stats    map[accounts.AccountID]stats.Stats
	failWith error
}

var _ stats.Store = (*Store)(nil)

func New() *Store {
	return &Store{stats: map[accounts.AccountID]stats.Stats{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Stats(_ context.Context, account accounts.AccountID) (stats.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return stats.Stats{}, s.failWith
	}
	counted, ok := s.stats[account]
	if !ok {
		return stats.Stats{}, stats.ErrNoStats
	}
	return counted, nil
}

func (s *Store) Record(_ context.Context, account accounts.AccountID, change func(stats.Stats) stats.Stats) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	counted, ok := s.stats[account]
	if !ok {
		counted = stats.NewStats(account)
	}
	s.stats[account] = change(counted)
	return nil
}

func (s *Store) StatsAfter(_ context.Context, after accounts.AccountID, limit int) ([]stats.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	var page []stats.Stats
	for account, stats := range s.stats {
		if bytes.Compare(account[:], after[:]) > 0 {
			page = append(page, stats)
		}
	}
	slices.SortFunc(page, func(a, b stats.Stats) int { return bytes.Compare(accountBytes(a), accountBytes(b)) })
	return page[:min(limit, len(page))], nil
}

func accountBytes(stats stats.Stats) []byte {
	account := stats.Account()
	return account[:]
}

func (s *Store) Forget(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.stats, account)
	return nil
}
