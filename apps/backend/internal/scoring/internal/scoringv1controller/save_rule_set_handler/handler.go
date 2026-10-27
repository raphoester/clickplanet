package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/save_rule_set_usecase"
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
	req *connect.Request[scoringv1.SaveRuleSetRequest],
) (*connect.Response[scoringv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, rules.Snapshot{Name: rules.SetName(set.GetName()), Rules: set.GetRules()})

	switch {
	case err == nil:
		return connect.NewResponse(&scoringv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, save_rule_set_usecase.ErrPlaying):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}
