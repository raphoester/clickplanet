package get_operator_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var ErrNoOperator = errors.New("this listener names no operator; the admin page calls the operator listener")

func New() GetOperatorHandler {
	return GetOperatorHandler{}
}

type GetOperatorHandler struct{}

func (GetOperatorHandler) GetOperator(
	ctx context.Context,
	_ *connect.Request[authv1.GetOperatorRequest],
) (*connect.Response[authv1.GetOperatorResponse], error) {
	operator := cpctx.GetOperator(ctx)
	if operator == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoOperator)
	}

	res := connect.NewResponse(&authv1.GetOperatorResponse{AccountId: operator})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
