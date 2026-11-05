package get_turn_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

type Query interface {
	Now(ctx context.Context) (*gamemasterv1.GetTurnResponse, error)
}

func New(query Query) GetTurnHandler {
	return GetTurnHandler{query: query}
}

type GetTurnHandler struct {
	query Query
}

func (h GetTurnHandler) GetTurn(
	ctx context.Context,
	_ *connect.Request[gamemasterv1.GetTurnRequest],
) (*connect.Response[gamemasterv1.GetTurnResponse], error) {
	now, err := h.query.Now(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(now), nil
}
