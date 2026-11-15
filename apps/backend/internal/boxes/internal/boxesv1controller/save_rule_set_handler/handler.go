package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type UseCase interface {
	Execute(ctx context.Context, in cprules.Snapshot[*boxesv1.Rules]) error
}

func New(useCase UseCase) SaveRuleSetHandler {
	return SaveRuleSetHandler{useCase: useCase}
}

type SaveRuleSetHandler struct {
	useCase UseCase
}

func (h SaveRuleSetHandler) SaveRuleSet(
	ctx context.Context,
	req *connect.Request[boxesv1.SaveRuleSetRequest],
) (*connect.Response[boxesv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, cprules.Snapshot[*boxesv1.Rules]{
		Name: cprules.SetName(set.GetName()), Rules: set.GetRules(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&boxesv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, cprules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
