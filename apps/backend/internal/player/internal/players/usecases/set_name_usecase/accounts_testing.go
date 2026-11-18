//go:build testing

package set_name_usecase

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type FakeAccounts struct {
	mu     sync.Mutex
	linked *cpcolls.Set[players.AccountID]
	asked  int
}

var _ Accounts = (*FakeAccounts)(nil)

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{linked: cpcolls.NewSet[players.AccountID]()}
}

func (f *FakeAccounts) Link(account players.AccountID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.linked.Add(account)
}

func (f *FakeAccounts) Linked(_ context.Context, account players.AccountID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked++
	return f.linked.Contains(account), nil
}

func (f *FakeAccounts) Asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.asked
}
