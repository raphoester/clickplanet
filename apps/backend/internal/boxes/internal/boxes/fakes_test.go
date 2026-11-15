package boxes

import (
	"context"
	"slices"
	"sync"
	"time"
)

// payer is a caller and when its account was made, as these tests click with.
type payer struct {
	Scope   string
	Account Account
	Linked  bool
	Created time.Time
}

func (p payer) caller() Caller {
	return Caller{Scope: p.Scope, Account: p.Account, Linked: p.Linked}
}

func entrantOf(p payer) Entrant {
	return EntrantOf(p.caller())
}

func clickOf(p payer) Click {
	return Click{Caller: p.caller(), Born: p.Created}
}

// fakeRoom is the wallet's word on room, as a test sets it: every hand has room until it is kept full.
type fakeRoom struct {
	mu      sync.Mutex
	full    map[Account]bool
	failing error
	asked   [][]Account

	// during runs while the room is asked, as another call into the registry would.
	during func()
}

func (f *fakeRoom) Any(_ context.Context, accounts []Account) (bool, error) {
	if f.during != nil {
		f.during()
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, slices.Clone(accounts))
	if f.failing != nil {
		return false, f.failing
	}

	for _, account := range accounts {
		if f.full[account] {
			return false, nil
		}
	}
	return true, nil
}

// Fill says the account's hand has room for no kind at all, or, false, room again.
func (f *fakeRoom) Fill(account Account, full bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.full == nil {
		f.full = map[Account]bool{}
	}
	f.full[account] = full
}

func (f *fakeRoom) Fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failing = err
}
