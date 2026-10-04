package get_me_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler/me_query"
)

type Query interface {
	Me(ctx context.Context, cookieHeader string) (*authv1.GetMeResponse, error)
}

func New(query Query) GetMeHandler {
	return GetMeHandler{query: query}
}

type GetMeHandler struct {
	query Query
}

func (h GetMeHandler) GetMe(
	ctx context.Context,
	req *connect.Request[authv1.GetMeRequest],
) (*connect.Response[authv1.GetMeResponse], error) {
	me, err := h.query.Me(ctx, req.Header().Get("Cookie"))
	if errors.Is(err, me_query.ErrNoAccount) {
		return nil, connect.NewError(connect.CodeUnauthenticated, me_query.ErrNoAccount)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the account: %w", err)
	}

	res := connect.NewResponse(me)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
