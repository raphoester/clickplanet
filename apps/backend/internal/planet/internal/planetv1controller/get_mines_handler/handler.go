package get_mines_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Query is the mines of the flag the caller plays for, and no other flag's.
type Query interface {
	MinesOf(ctx context.Context, payer clicks.Payer) (*planetv1.GetMinesResponse, error)
}

func New(query Query) GetMinesHandler {
	return GetMinesHandler{query: query}
}

type GetMinesHandler struct {
	query Query
}

func (h GetMinesHandler) GetMines(
	ctx context.Context,
	_ *connect.Request[planetv1.GetMinesRequest],
) (*connect.Response[planetv1.GetMinesResponse], error) {
	mines, err := h.query.MinesOf(ctx, clicks.PayerOf(ctx))
	if err != nil {
		return nil, err //nolint:wrapcheck // the query names it, and the error net answers it.
	}

	return connect.NewResponse(mines), nil
}
