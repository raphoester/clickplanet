//go:build testing

package players

import (
	"context"
	"sync"
	"time"
)

type FakeAccounts struct {
	mu       sync.Mutex
	created  map[AccountID]time.Time
	failWith error
}

func NewFakeAccounts() *FakeAccounts {
	return &FakeAccounts{created: map[AccountID]time.Time{}}
}

func (f *FakeAccounts) Create(account AccountID, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.created[account] = at
}

func (f *FakeAccounts) FailWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failWith = err
}

func (f *FakeAccounts) CreatedAt(_ context.Context, account AccountID) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return time.Time{}, f.failWith
	}
	return f.created[account], nil
}

func (f *FakeAccounts) CreationDates(_ context.Context, accounts []AccountID) (map[AccountID]time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failWith != nil {
		return nil, f.failWith
	}
	dates := map[AccountID]time.Time{}
	for _, account := range accounts {
		if at, ok := f.created[account]; ok {
			dates[account] = at
		}
	}
	return dates, nil
}
