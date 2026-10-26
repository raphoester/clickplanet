package get_turn_handler

import (
	"context"

	"connectrpc.com/connect"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
)

type Query interface {
	Turn(ctx context.Context) (*metagamev1.GetTurnResponse, error)
}

func New(query Query) GetTurnHandler {
	return GetTurnHandler{query: query}
}

type GetTurnHandler struct {
	query Query
}

func (h GetTurnHandler) GetTurn(
	ctx context.Context,
	_ *connect.Request[metagamev1.GetTurnRequest],
) (*connect.Response[metagamev1.GetTurnResponse], error) {
	answer, err := h.query.Turn(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(answer), nil
}
