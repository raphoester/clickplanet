//go:build testing

package titles

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type FakeAccounts struct {
	mu       sync.Mutex
	created  map[players.AccountID]time.Time
	failWith error
}

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{created: map[players.AccountID]time.Time{}}
}

func (f *FakeAccounts) Create(account players.AccountID, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.created[account] = at
}

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

func (f *FakeAccounts) CreationDates(_ context.Context, accounts []players.AccountID) (map[players.AccountID]time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return nil, f.failWith
	}
	dates := map[players.AccountID]time.Time{}
	for _, account := range accounts {
		if at, ok := f.created[account]; ok {
			dates[account] = at
		}
	}
	return dates, nil
}
