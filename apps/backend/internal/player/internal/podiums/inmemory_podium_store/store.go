//go:build testing

package inmemory_podium_store

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
)

type Store struct {
	mu       sync.Mutex
	kept     map[players.AccountID][]podiums.PlaceSnapshot
	failWith error
}

var _ podiums.Store = (*Store)(nil)

func New() *Store {
	return &Store{kept: map[players.AccountID][]podiums.PlaceSnapshot{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Record(_ context.Context, places []podiums.Place) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for _, place := range places {
		snapshot := place.TakeSnapshot()
		kept := slices.DeleteFunc(s.kept[snapshot.Account], func(each podiums.PlaceSnapshot) bool {
			return each.Season == snapshot.Season
		})
		kept = append(kept, snapshot)
		slices.SortFunc(kept, func(a, b podiums.PlaceSnapshot) int { return int(a.Season) - int(b.Season) })
		s.kept[snapshot.Account] = kept
	}
	return nil
}

func (s *Store) Places(_ context.Context, accounts []players.AccountID) (map[players.AccountID]podiums.Places, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	found := map[players.AccountID]podiums.Places{}
	for _, account := range accounts {
		for _, snapshot := range s.kept[account] {
			place, err := snapshot.Restore()
			if err != nil {
				return nil, fmt.Errorf("failed to restore a podium place: %w", err)
			}
			found[account] = append(found[account], place)
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
