//go:build testing

package subscriptions

import (
	"context"
	"errors"
	"slices"
	"sync"
)

var ErrFakeAudienceDown = errors.New("the fake audience was told to fail")

type Call struct {
	Method  string
	Address Address
}

type FakeAudience struct {
	mu        sync.Mutex
	calls     []Call
	confirmed []Address
	failing   []string
}

var _ Audience = (*FakeAudience)(nil)

func (f *FakeAudience) Confirm(address Address) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.confirmed = append(f.confirmed, address)
}

func (f *FakeAudience) Fail(methods ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failing = methods
}

func (f *FakeAudience) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.calls)
}

func (f *FakeAudience) Join(_ context.Context, address Address) error {
	return f.record("Join", address)
}

func (f *FakeAudience) Invite(_ context.Context, address Address) error {
	return f.record("Invite", address)
}

func (f *FakeAudience) Joined(_ context.Context, address Address) (bool, error) {
	if err := f.record("Joined", address); err != nil {
		return false, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Contains(f.confirmed, address), nil
}

func (f *FakeAudience) Leave(_ context.Context, address Address) error {
	return f.record("Leave", address)
}

func (f *FakeAudience) Forget(_ context.Context, address Address) error {
	return f.record("Forget", address)
}

func (f *FakeAudience) record(method string, address Address) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, Call{Method: method, Address: address})
	if slices.Contains(f.failing, method) {
		return ErrFakeAudienceDown
	}
	return nil
}
