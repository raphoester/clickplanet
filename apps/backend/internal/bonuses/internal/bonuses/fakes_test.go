package bonuses

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

// fakeRoom is the wallet's word on room, as a test sets it: every account has room for every kind until it is kept full.
type fakeRoom struct {
	mu      sync.Mutex
	full    map[Account][]Kind
	failing error
	asked   [][]Account

	// during runs while the room is asked, as another call into the registry would.
	during func()
}

func (f *fakeRoom) Full(_ context.Context, accounts []Account) (map[Account][]Kind, error) {
	if f.during != nil {
		f.during()
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked = append(f.asked, slices.Clone(accounts))
	if f.failing != nil {
		return nil, f.failing
	}

	full := make(map[Account][]Kind, len(accounts))
	for _, account := range accounts {
		full[account] = f.full[account]
	}

	return full, nil
}

// Keep says the account has no room for these kinds, and room for every other.
func (f *fakeRoom) Keep(account Account, kinds ...Kind) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.full == nil {
		f.full = map[Account][]Kind{}
	}
	f.full[account] = kinds
}

func (f *fakeRoom) Fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failing = err
}
