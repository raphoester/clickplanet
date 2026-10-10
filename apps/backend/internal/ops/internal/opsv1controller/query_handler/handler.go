package query_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query"
)

type Query interface {
	Rows(ctx context.Context, statement string, limit uint32) (*opsv1.QueryResponse, error)
}

func New(query Query) QueryHandler {
	return QueryHandler{query: query}
}

type QueryHandler struct {
	query Query
}

func (h QueryHandler) Query(
	ctx context.Context,
	req *connect.Request[opsv1.QueryRequest],
) (*connect.Response[opsv1.QueryResponse], error) {
	rows, err := h.query.Rows(ctx, req.Msg.GetStatement(), req.Msg.GetLimit())

	switch {
	case errors.Is(err, statement_query.ErrNoStatement),
		errors.Is(err, statement_query.ErrLimitTooHigh),
		errors.Is(err, statement_query.ErrRefused):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, statement_query.ErrTooSlow):
		return nil, connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, statement_query.ErrUnreachable):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // a failure nobody named is the error net's to answer.
	}

	return connect.NewResponse(rows), nil
}
