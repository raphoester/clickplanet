//go:build testing

package set_name_usecase

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// FakeAccounts plays the auth module: every account is a guest until it is Linked, and it can fail on demand.
type FakeAccounts struct {
	mu       sync.Mutex
	linked   *cpcolls.Set[players.AccountID]
	failWith error
	asked    int
}

var _ Accounts = (*FakeAccounts)(nil)

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{linked: cpcolls.NewSet[players.AccountID]()}
}

// Link makes the account one signed in with a provider.
func (f *FakeAccounts) Link(account players.AccountID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.linked.Add(account)
}

// FailWith makes every later call answer err.
func (f *FakeAccounts) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakeAccounts) Linked(_ context.Context, account players.AccountID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked++
	if f.failWith != nil {
		return false, f.failWith
	}
	return f.linked.Contains(account), nil
}

// Asked is how many times the fake was called.
func (f *FakeAccounts) Asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.asked
}
