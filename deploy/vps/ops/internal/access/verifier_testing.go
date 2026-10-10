//go:build testing

package access

import (
	"context"
	"errors"
)

var ErrUnknownAssertion = errors.New("unknown assertion")

type StaticVerifier map[string]Caller

func (s StaticVerifier) Verify(_ context.Context, assertion string) (Caller, error) {
	caller, known := s[assertion]
	if !known {
		return "", ErrUnknownAssertion
	}
	return caller, nil
}
