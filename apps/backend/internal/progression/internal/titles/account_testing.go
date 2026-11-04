//go:build testing

package titles

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type FakeAccounts struct {
	mu       sync.Mutex
	accounts map[accounts.AccountID]accounts.Account
	failWith error
}

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{accounts: map[accounts.AccountID]accounts.Account{}}
}

func (f *FakeAccounts) Create(id accounts.AccountID, account accounts.Account) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.accounts[id] = account
}

func (f *FakeAccounts) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakeAccounts) Account(_ context.Context, id accounts.AccountID) (accounts.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return accounts.Account{}, f.failWith
	}
	return f.accounts[id], nil
}

func (f *FakeAccounts) Accounts(_ context.Context, ids []accounts.AccountID) (map[accounts.AccountID]accounts.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return nil, f.failWith
	}
	found := map[accounts.AccountID]accounts.Account{}
	for _, id := range ids {
		if account, ok := f.accounts[id]; ok {
			found[id] = account
		}
	}
	return found, nil
}
