package get_operator_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
)

type Query interface {
	Operator(ctx context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error)
}

func New(query Query) GetOperatorHandler {
	return GetOperatorHandler{query: query}
}

type GetOperatorHandler struct {
	query Query
}

func (h GetOperatorHandler) GetOperator(
	ctx context.Context,
	req *connect.Request[authv1.GetOperatorRequest],
) (*connect.Response[authv1.GetOperatorResponse], error) {
	operator, err := h.query.Operator(ctx, req.Msg.GetCookie())

	switch {
	case err == nil:
		return connect.NewResponse(operator), nil
	case errors.Is(err, operator_query.ErrNoSession):
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operator_query.ErrNotAnOperator):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	return nil, fmt.Errorf("failed to read the operator: %w", err)
}
