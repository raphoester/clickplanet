//go:build testing

package inmemory_season_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
)

type Store struct {
	mu       sync.Mutex
	records  map[players.AccountID]seasons.RecordSnapshot
	failWith error
}

var _ seasons.Store = (*Store)(nil)

func New() *Store {
	return &Store{records: map[players.AccountID]seasons.RecordSnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Record(_ context.Context, account players.AccountID) (seasons.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return seasons.Record{}, s.failWith
	}
	if kept, ok := s.records[account]; ok {
		return kept.Restore(), nil
	}
	return seasons.NewRecord(account), nil
}

func (s *Store) Records(_ context.Context, accounts []players.AccountID) (map[players.AccountID]seasons.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	records := map[players.AccountID]seasons.Record{}
	for _, account := range accounts {
		if kept, ok := s.records[account]; ok {
			records[account] = kept.Restore()
		}
	}
	return records, nil
}

func (s *Store) Save(_ context.Context, record seasons.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	snapshot := record.TakeSnapshot()
	if len(snapshot.Seasons) == 0 {
		delete(s.records, snapshot.Account)
		return nil
	}
	s.records[snapshot.Account] = snapshot
	return nil
}
