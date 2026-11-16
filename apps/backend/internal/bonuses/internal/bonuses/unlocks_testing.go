//go:build testing

package bonuses

import (
	"context"
	"sync"
)

// FakeUnlocks stands for the kinds progression unlocked, in the bonuses module's tests: what it is told for an account,
// none for the others.
type FakeUnlocks struct {
	mu       sync.Mutex
	unlocked map[Account]Unlocked
	failing  error
}

func NewFakeUnlocks() *FakeUnlocks {
	return &FakeUnlocks{unlocked: map[Account]Unlocked{}}
}

func (u *FakeUnlocks) Unlock(account Account, kinds ...Kind) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.unlocked[account] = kinds
}

func (u *FakeUnlocks) FailWith(err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.failing = err
}

func (u *FakeUnlocks) Unlocked(_ context.Context, account Account) (Unlocked, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.failing != nil {
		return nil, u.failing
	}
	return u.unlocked[account], nil
}
