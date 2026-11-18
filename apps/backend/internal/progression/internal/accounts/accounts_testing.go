//go:build testing

package accounts

import (
	"context"
	"slices"
	"sync"
)

type FakeAccounts struct {
	mu       sync.Mutex
	accounts map[AccountID]Account
	asked    []AccountID
}

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{accounts: map[AccountID]Account{}}
}

func (f *FakeAccounts) Create(id AccountID, account Account) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.accounts[id] = account
}

func (f *FakeAccounts) Asked() []AccountID {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.asked)
}

func (f *FakeAccounts) Account(_ context.Context, id AccountID) (Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, id)
	return f.accounts[id], nil
}

func (f *FakeAccounts) Accounts(_ context.Context, ids []AccountID) (map[AccountID]Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, ids...)
	found := map[AccountID]Account{}
	for _, id := range ids {
		if account, ok := f.accounts[id]; ok {
			found[id] = account
		}
	}
	return found, nil
}
