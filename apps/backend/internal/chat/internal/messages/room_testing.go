//go:build testing

package messages

import (
	"context"
	"errors"
)

// The flag each account plays for; an account it does not name plays for none.
type FakeFlags map[AccountID]string

func (f FakeFlags) Flag(_ context.Context, account AccountID) (string, error) {
	return f[account], nil
}

var ErrFlagsDown = errors.New("planet does not answer")

type FailingFlags struct{}

func (FailingFlags) Flag(context.Context, AccountID) (string, error) {
	return "", ErrFlagsDown
}
