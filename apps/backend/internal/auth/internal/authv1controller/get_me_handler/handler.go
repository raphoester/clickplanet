// Package get_me_handler serves auth.v1.AuthService/GetMe.
package get_me_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type UseCase interface {
	Execute(ctx context.Context, cookieHeader string) (uuid.UUID, error)
}

func New(useCase UseCase) GetMeHandler {
	return GetMeHandler{useCase: useCase}
}

type GetMeHandler struct {
	useCase UseCase
}

func (h GetMeHandler) GetMe(
	ctx context.Context,
	req *connect.Request[authv1.GetMeRequest],
) (*connect.Response[authv1.GetMeResponse], error) {
	account, err := h.useCase.Execute(ctx, req.Header().Get("Cookie"))
	if errors.Is(err, accounts.ErrNoAccount) {
		return nil, connect.NewError(connect.CodeUnauthenticated, accounts.ErrNoAccount)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the account: %w", err)
	}

	res := connect.NewResponse(&authv1.GetMeResponse{AccountId: account.String()})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
