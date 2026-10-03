//go:build testing

package titles

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type FakeAccounts struct {
	mu       sync.Mutex
	accounts map[players.AccountID]players.Account
	failWith error
}

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{accounts: map[players.AccountID]players.Account{}}
}

func (f *FakeAccounts) Create(id players.AccountID, account players.Account) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.accounts[id] = account
}

func (f *FakeAccounts) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakeAccounts) Account(_ context.Context, id players.AccountID) (players.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return players.Account{}, f.failWith
	}
	return f.accounts[id], nil
}

func (f *FakeAccounts) Accounts(_ context.Context, ids []players.AccountID) (map[players.AccountID]players.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return nil, f.failWith
	}
	found := map[players.AccountID]players.Account{}
	for _, id := range ids {
		if account, ok := f.accounts[id]; ok {
			found[id] = account
		}
	}
	return found, nil
}
