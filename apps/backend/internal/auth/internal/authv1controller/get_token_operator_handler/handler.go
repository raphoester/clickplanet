package get_token_operator_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_token_operator_handler/token_operator_query"
)

type Query interface {
	Operator(ctx context.Context, token string) (*authv1.GetTokenOperatorResponse, error)
}

func New(query Query) GetTokenOperatorHandler {
	return GetTokenOperatorHandler{query: query}
}

type GetTokenOperatorHandler struct {
	query Query
}

func (h GetTokenOperatorHandler) GetTokenOperator(
	ctx context.Context,
	req *connect.Request[authv1.GetTokenOperatorRequest],
) (*connect.Response[authv1.GetTokenOperatorResponse], error) {
	operator, err := h.query.Operator(ctx, req.Msg.GetToken())

	switch {
	case err == nil:
		return connect.NewResponse(operator), nil
	case errors.Is(err, token_operator_query.ErrNoToken):
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, token_operator_query.ErrNotAnOperator):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	return nil, fmt.Errorf("failed to read the token's operator: %w", err)
}
