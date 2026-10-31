package get_round_players_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_players_handler/round_players_query"
)

type Query interface {
	Players(ctx context.Context, id books.RoundID) ([]*scoringv1.RoundPlayer, error)
}

func New(query Query) GetRoundPlayersHandler {
	return GetRoundPlayersHandler{query: query}
}

type GetRoundPlayersHandler struct {
	query Query
}

func (h GetRoundPlayersHandler) GetRoundPlayers(
	ctx context.Context,
	req *connect.Request[scoringv1.GetRoundPlayersRequest],
) (*connect.Response[scoringv1.GetRoundPlayersResponse], error) {
	id, err := books.ParseRoundID(req.Msg.GetRoundId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	players, err := h.query.Players(ctx, id)
	if errors.Is(err, round_players_query.ErrNoRound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&scoringv1.GetRoundPlayersResponse{Players: players}), nil
}
