package get_round_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query"
)

type Query interface {
	Round(ctx context.Context, id books.RoundID) (*scoringv1.Round, error)
}

func New(query Query) GetRoundHandler {
	return GetRoundHandler{query: query}
}

// GetRoundHandler serves both ScoringService and InternalService: they answer the same round.
type GetRoundHandler struct {
	query Query
}

func (h GetRoundHandler) GetRound(
	ctx context.Context,
	req *connect.Request[scoringv1.GetRoundRequest],
) (*connect.Response[scoringv1.GetRoundResponse], error) {
	id, err := books.ParseRoundID(req.Msg.GetRoundId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	round, err := h.query.Round(ctx, id)
	if errors.Is(err, round_query.ErrNoRound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.GetRoundResponse{Round: round}), nil
}
