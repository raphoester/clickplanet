package get_book_history_handler

import (
	"context"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Query interface {
	History(ctx context.Context, book books.BookID) ([]*scoringv1.BookStep, error)
}

func New(query Query) GetBookHistoryHandler {
	return GetBookHistoryHandler{query: query}
}

type GetBookHistoryHandler struct {
	query Query
}

func (h GetBookHistoryHandler) GetBookHistory(
	ctx context.Context,
	req *connect.Request[scoringv1.GetBookHistoryRequest],
) (*connect.Response[scoringv1.GetBookHistoryResponse], error) {
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	steps, err := h.query.History(ctx, book)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.GetBookHistoryResponse{Steps: steps}), nil
}
