package save_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/rule_set_message"
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
	err := h.useCase.Execute(ctx, rule_set_message.Decode(req.Msg.GetSet()))

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.SaveRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
