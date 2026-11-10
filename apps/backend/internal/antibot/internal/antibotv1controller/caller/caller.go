package caller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrNoAccount = errors.New("this procedure answers for an account; mint with auth.v1.AuthService/CreateSession")

func AccountOf(ctx context.Context) (string, error) {
	account, err := uuid.Parse(cpctx.GetAccount(ctx))
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}
	return account.String(), nil
}
