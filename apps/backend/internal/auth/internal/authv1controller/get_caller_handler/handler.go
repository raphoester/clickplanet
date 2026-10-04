package get_caller_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
)

type Query interface {
	Caller(ctx context.Context, cookieHeader string) (*authv1.GetCallerResponse, error)
}

func New(query Query) GetCallerHandler {
	return GetCallerHandler{query: query}
}

type GetCallerHandler struct {
	query Query
}

func (h GetCallerHandler) GetCaller(
	ctx context.Context,
	req *connect.Request[authv1.GetCallerRequest],
) (*connect.Response[authv1.GetCallerResponse], error) {
	caller, err := h.query.Caller(ctx, req.Msg.GetCookie())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(caller), nil
}
