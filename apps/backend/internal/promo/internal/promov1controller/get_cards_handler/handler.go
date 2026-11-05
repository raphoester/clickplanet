package get_cards_handler

import (
	"context"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
)

type Query interface {
	Cards(ctx context.Context) (*promov1.GetCardsResponse, error)
}

func New(query Query) GetCardsHandler {
	return GetCardsHandler{query: query}
}

type GetCardsHandler struct {
	query Query
}

func (h GetCardsHandler) GetCards(
	ctx context.Context,
	_ *connect.Request[promov1.GetCardsRequest],
) (*connect.Response[promov1.GetCardsResponse], error) {
	answer, err := h.query.Cards(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(answer), nil
}
