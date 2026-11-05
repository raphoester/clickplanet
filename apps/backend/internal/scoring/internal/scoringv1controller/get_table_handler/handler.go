package get_table_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler/scores_query"
)

type Query interface {
	Table(ctx context.Context, book books.BookID) (*scoringv1.GetTableResponse, error)
}

func New(query Query) GetTableHandler {
	return GetTableHandler{query: query}
}

type GetTableHandler struct {
	query Query
}

func (h GetTableHandler) GetTable(
	ctx context.Context,
	req *connect.Request[scoringv1.GetTableRequest],
) (*connect.Response[scoringv1.GetTableResponse], error) {
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	table, err := h.query.Table(ctx, book)
	if errors.Is(err, scores_query.ErrNoBook) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(table), nil
}
