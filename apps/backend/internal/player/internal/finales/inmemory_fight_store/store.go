//go:build testing

package inmemory_fight_store

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store struct {
	mu       sync.Mutex
	kept     map[players.AccountID][]finales.FightSnapshot
	failWith error
}

var _ finales.Store = (*Store)(nil)

func New() *Store {
	return &Store{kept: map[players.AccountID][]finales.FightSnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Record(_ context.Context, fights []finales.Fight) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for _, fight := range fights {
		snapshot := fight.TakeSnapshot()
		kept := s.kept[snapshot.Account]
		if slices.ContainsFunc(kept, func(each finales.FightSnapshot) bool { return each.Season == snapshot.Season }) {
			continue
		}
		kept = append(kept, snapshot)
		slices.SortFunc(kept, func(a, b finales.FightSnapshot) int { return int(a.Season) - int(b.Season) })
		s.kept[snapshot.Account] = kept
	}
	return nil
}

func (s *Store) Fights(_ context.Context, accounts []players.AccountID) (map[players.AccountID]finales.Fights, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	found := map[players.AccountID]finales.Fights{}
	for _, account := range accounts {
		for _, snapshot := range s.kept[account] {
			fight, err := snapshot.Restore()
			if err != nil {
				return nil, fmt.Errorf("failed to restore a fight: %w", err)
			}
			found[account] = append(found[account], fight)
		}
	}
	return found, nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.kept, account)
	return nil
}
