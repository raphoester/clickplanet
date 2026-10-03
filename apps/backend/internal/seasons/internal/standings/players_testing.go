//go:build testing

package standings

import (
	"context"
	"sync"
)

type FakePlayers struct {
	mu       sync.Mutex
	players  map[AccountID]Player
	asked    int
	failWith error
}

func NewFakePlayers() *FakePlayers {
	return &FakePlayers{players: map[AccountID]Player{}}
}

func (f *FakePlayers) Add(account AccountID, player Player) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.players[account] = player
}

func (f *FakePlayers) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakePlayers) Asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.asked
}

func (f *FakePlayers) Players(_ context.Context, accounts []AccountID) (map[AccountID]Player, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked++
	if f.failWith != nil {
		return nil, f.failWith
	}
	found := map[AccountID]Player{}
	for _, account := range accounts {
		if player, ok := f.players[account]; ok {
			found[account] = player
		}
	}
	return found, nil
}
