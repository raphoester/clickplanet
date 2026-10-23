package pledge_allegiance_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
)

type Book interface {
	Pledge(ctx context.Context, payer clicks.Payer, flag string) error
}

func New(book Book) PledgeAllegianceHandler {
	return PledgeAllegianceHandler{book: book}
}

type PledgeAllegianceHandler struct {
	book Book
}

func (h PledgeAllegianceHandler) PledgeAllegiance(
	ctx context.Context,
	req *connect.Request[planetv1.PledgeAllegianceRequest],
) (*connect.Response[planetv1.PledgeAllegianceResponse], error) {
	err := h.book.Pledge(ctx, clicks.PayerOf(ctx), req.Msg.GetCountryId())
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.PledgeAllegianceResponse{}), nil
	case errors.Is(err, allegiances.ErrNoAccount):
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, clicks.ErrUnknownCountry):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
}
