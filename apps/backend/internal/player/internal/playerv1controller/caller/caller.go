// Package caller reads who calls a PlayerService procedure: the account the session interceptor put on the context.
package caller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// ErrNoAccount is a caller whose token names no account: the deprecated mint, or no token at all.
var ErrNoAccount = errors.New("this procedure answers for an account; mint with auth.v1.AuthService/CreateSession")

// AccountOf is the caller's account, or CodeUnauthenticated.
func AccountOf(ctx context.Context) (players.AccountID, error) {
	account, err := players.AccountIDOf(cpctx.GetAccount(ctx))
	if err != nil {
		return players.AccountID{}, connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}
	return account, nil
}
