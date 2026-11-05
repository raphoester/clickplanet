package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
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
	req *connect.Request[antibotv1.SaveRuleSetRequest],
) (*connect.Response[antibotv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, rules.Snapshot{Name: rules.SetName(set.GetName()), Rules: set.GetRules()})

	switch {
	case err == nil:
		return connect.NewResponse(&antibotv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
