//go:build testing

package messages

import (
	"context"
	"errors"
)

// The flag each account plays for; an account it does not name plays for none.
type FakeFlags map[AccountID]string

func (f FakeFlags) Flag(_ context.Context, account AccountID) (Pledge, error) {
	return Pledge{Flag: f[account]}, nil
}

// FakeClosedFlags is the flag each account plays for, every one of them closed.
type FakeClosedFlags map[AccountID]string

func (f FakeClosedFlags) Flag(_ context.Context, account AccountID) (Pledge, error) {
	return Pledge{Flag: f[account], Closed: true}, nil
}

var ErrFlagsDown = errors.New("planet does not answer")

type FailingFlags struct{}

func (FailingFlags) Flag(context.Context, AccountID) (Pledge, error) {
	return Pledge{}, ErrFlagsDown
}
