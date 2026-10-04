package set_rules_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in set_rules_usecase.In) error
}

func New(useCase UseCase) SetRulesHandler {
	return SetRulesHandler{useCase: useCase}
}

type SetRulesHandler struct {
	useCase UseCase
}

func (h SetRulesHandler) SetRules(
	ctx context.Context,
	req *connect.Request[planetv1.SetRulesRequest],
) (*connect.Response[planetv1.SetRulesResponse], error) {
	in := set_rules_usecase.In{
		RefillMultiplier: req.Msg.GetRefillMultiplier(),
		BoxInterval:      time.Duration(req.Msg.GetBoxIntervalMs()) * time.Millisecond,
		Frozen:           req.Msg.GetFrozen(),
	}
	if gift := req.Msg.GetGift(); gift != nil {
		in.GiftTag = tempo.GiftTag(gift.GetTag())
		if before := gift.GetAccountsMadeBeforeUnixMs(); before != 0 {
			in.GiftMadeBefore = time.UnixMilli(before).UTC()
		}
	}

	err := h.useCase.Execute(ctx, in)
	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.SetRulesResponse{}), nil
	case errors.Is(err, tempo.ErrInvalidRules):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
