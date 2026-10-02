package caller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrNoAccount = errors.New("this procedure answers for an account; mint with auth.v1.AuthService/CreateSession")

func AccountOf(ctx context.Context) (players.AccountID, error) {
	account, err := players.AccountIDOf(cpctx.GetAccount(ctx))
	if err != nil {
		return players.AccountID{}, connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}
	return account, nil
}
