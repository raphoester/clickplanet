package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type UseCase interface {
	Execute(ctx context.Context, in rules.Snapshot) error
}

func New(useCase UseCase) SaveRuleSetHandler {
	return SaveRuleSetHandler{useCase: useCase}
}

type SaveRuleSetHandler struct {
	useCase UseCase
}

func (h SaveRuleSetHandler) SaveRuleSet(
	ctx context.Context,
	req *connect.Request[planetv1.SaveRuleSetRequest],
) (*connect.Response[planetv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, rules.Snapshot{
		Name:  rules.SetName(set.GetName()),
		Tempo: set.GetTempo(),
		Boxes: bonuses.Pace{
			MinInterval:       set.GetBoxes().GetMinInterval().AsDuration(),
			MaxInterval:       set.GetBoxes().GetMaxInterval().AsDuration(),
			MaxChargesPerHour: int(set.GetBoxes().GetMaxChargesPerHour()),
		},
		Crowd: int(set.GetAntiBot().GetCrowd()),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
