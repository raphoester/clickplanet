package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
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
	req *connect.Request[bonusesv1.SaveRuleSetRequest],
) (*connect.Response[bonusesv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, rules.Snapshot{Name: rules.SetName(set.GetName()), Rules: set.GetRules()})

	switch {
	case err == nil:
		return connect.NewResponse(&bonusesv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
