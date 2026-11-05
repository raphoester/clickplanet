package get_scores_handler

import (
	"context"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Query interface {
	Scores(ctx context.Context, book books.BookID) (*scoringv1.Scores, error)
}

func New(query Query) GetScoresHandler {
	return GetScoresHandler{query: query}
}

type GetScoresHandler struct {
	query Query
}

func (h GetScoresHandler) GetScores(
	ctx context.Context,
	req *connect.Request[scoringv1.GetScoresRequest],
) (*connect.Response[scoringv1.GetScoresResponse], error) {
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	scores, err := h.query.Scores(ctx, book)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.GetScoresResponse{Scores: scores}), nil
}
