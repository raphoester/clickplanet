package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type UseCase interface {
	Execute(ctx context.Context, in cprules.Snapshot[*quizzesv1.Rules]) error
}

func New(useCase UseCase) SaveRuleSetHandler {
	return SaveRuleSetHandler{useCase: useCase}
}

type SaveRuleSetHandler struct {
	useCase UseCase
}

func (h SaveRuleSetHandler) SaveRuleSet(
	ctx context.Context,
	req *connect.Request[quizzesv1.SaveRuleSetRequest],
) (*connect.Response[quizzesv1.SaveRuleSetResponse], error) {
	set := req.Msg.GetSet()
	err := h.useCase.Execute(ctx, cprules.Snapshot[*quizzesv1.Rules]{
		Name: cprules.SetName(set.GetName()), Rules: set.GetRules(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&quizzesv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, cprules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
