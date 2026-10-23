package pledge_allegiance_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
)

type Book interface {
	Pledge(ctx context.Context, payer clicks.Payer, flag string) error
}

type Query interface {
	Allegiance(ctx context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error)
}

func New(book Book, query Query) PledgeAllegianceHandler {
	return PledgeAllegianceHandler{book: book, query: query}
}

type PledgeAllegianceHandler struct {
	book  Book
	query Query
}

func (h PledgeAllegianceHandler) PledgeAllegiance(
	ctx context.Context,
	req *connect.Request[planetv1.PledgeAllegianceRequest],
) (*connect.Response[planetv1.PledgeAllegianceResponse], error) {
	payer := clicks.PayerOf(ctx)

	err := h.book.Pledge(ctx, payer, req.Msg.GetCountryId())
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	held, err := h.query.Allegiance(ctx, allegiances.AccountOf(payer))
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.PledgeAllegianceResponse{Allegiance: held.GetAllegiance()}), nil
}
