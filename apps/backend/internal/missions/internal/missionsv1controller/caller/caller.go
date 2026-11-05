package caller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrNoAccount = errors.New("this procedure answers for an account; mint with auth.v1.AuthService/CreateSession")

func AccountOf(ctx context.Context) (missions.AccountID, error) {
	account, err := missions.AccountIDOf(cpctx.GetAccount(ctx))
	if err != nil {
		return missions.AccountID{}, connect.NewError(connect.CodeUnauthenticated, ErrNoAccount)
	}
	return account, nil
}
