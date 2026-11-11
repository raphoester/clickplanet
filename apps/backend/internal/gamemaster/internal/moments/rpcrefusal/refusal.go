// Package rpcrefusal reads a module's answer to a call: FailedPrecondition and InvalidArgument refuse it for good, any
// other error is no answer.
package rpcrefusal

import (
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

// Of is the error as the gamemaster reads it: moments.ErrRefused for a refusal for good.
func Of(err error) error {
	if code := connect.CodeOf(err); code == connect.CodeFailedPrecondition || code == connect.CodeInvalidArgument {
		return fmt.Errorf("%w: %w", moments.ErrRefused, err)
	}
	return err
}
