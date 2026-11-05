package caller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrNoAccount = errors.New("this procedure answers for an account; mint with auth.v1.AuthService/CreateSession")

func AccountOf(ctx context.Context) (casts.AccountID, error) {
	account, err := casts.AccountIDOf(cpctx.GetAccount(ctx))
	if err != nil {
		return casts.AccountID{}, connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}
	return account, nil
}
