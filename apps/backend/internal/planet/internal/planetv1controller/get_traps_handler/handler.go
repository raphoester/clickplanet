package get_traps_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Query is the traps of the flag the caller plays for, and no other flag's.
type Query interface {
	TrapsOf(ctx context.Context, payer clicks.Payer) (*planetv1.GetTrapsResponse, error)
}

func New(query Query) GetTrapsHandler {
	return GetTrapsHandler{query: query}
}

type GetTrapsHandler struct {
	query Query
}

func (h GetTrapsHandler) GetTraps(
	ctx context.Context,
	_ *connect.Request[planetv1.GetTrapsRequest],
) (*connect.Response[planetv1.GetTrapsResponse], error) {
	traps, err := h.query.TrapsOf(ctx, clicks.PayerOf(ctx))
	if err != nil {
		return nil, err //nolint:wrapcheck // the query names it, and the error net answers it.
	}

	return connect.NewResponse(traps), nil
}
