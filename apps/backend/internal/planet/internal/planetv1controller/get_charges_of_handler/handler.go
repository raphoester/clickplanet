package get_charges_of_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

type Hands interface {
	Held(ctx context.Context, holder charges.Holder) (charges.Held, error)
}

func New(hands Hands) GetChargesOfHandler {
	return GetChargesOfHandler{hands: hands}
}

// GetChargesOfHandler answers all an account holds now, for another module: the hand a wrong answer leaves.
type GetChargesOfHandler struct {
	hands Hands
}

func (h GetChargesOfHandler) GetChargesOf(
	ctx context.Context, req *connect.Request[planetv1.GetChargesOfRequest],
) (*connect.Response[planetv1.GetChargesOfResponse], error) {
	holder, err := charges.ParseHolder(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := h.hands.Held(ctx, holder)
	if err != nil {
		return nil, err //nolint:wrapcheck // the wallet named it.
	}
	return connect.NewResponse(&planetv1.GetChargesOfResponse{Charges: chargesheld.Encode(held)}), nil
}
