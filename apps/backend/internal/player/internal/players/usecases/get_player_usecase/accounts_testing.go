//go:build testing

package get_player_usecase

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// FakeAccounts plays the auth module: an account it was not told of is unknown, and it can fail on demand.
type FakeAccounts struct {
	mu       sync.Mutex
	created  map[players.AccountID]time.Time
	failWith error
}

var _ Accounts = (*FakeAccounts)(nil)

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{created: map[players.AccountID]time.Time{}}
}

// Create makes the account at at.
func (f *FakeAccounts) Create(account players.AccountID, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.created[account] = at
}

// FailWith makes every later call answer err.
func (f *FakeAccounts) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakeAccounts) CreatedAt(_ context.Context, account players.AccountID) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return time.Time{}, f.failWith
	}
	return f.created[account], nil
}
