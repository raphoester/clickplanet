package get_board_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
)

type Query interface {
	Board(ctx context.Context, view board_query.View) (*scoringv1.GetBoardResponse, error)
}

func New(query Query) GetBoardHandler {
	return GetBoardHandler{query: query}
}

type GetBoardHandler struct {
	query Query
}

func (h GetBoardHandler) GetBoard(
	ctx context.Context,
	req *connect.Request[scoringv1.GetBoardRequest],
) (*connect.Response[scoringv1.GetBoardResponse], error) {
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	board, err := h.query.Board(ctx, board_query.View{Book: book, Country: req.Msg.GetCountryId()})
	if errors.Is(err, board_query.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(board), nil
}
