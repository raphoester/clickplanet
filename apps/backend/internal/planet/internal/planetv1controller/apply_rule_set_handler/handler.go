package apply_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in apply_rule_set_usecase.In) error
}

// New answers for one source: the internal service's handler keeps internal changes, the admin service's an operator's.
func New(useCase UseCase, source rules.Source) ApplyRuleSetHandler {
	return ApplyRuleSetHandler{useCase: useCase, source: source}
}

type ApplyRuleSetHandler struct {
	useCase UseCase
	source  rules.Source
}

func (h ApplyRuleSetHandler) ApplyRuleSet(
	ctx context.Context,
	req *connect.Request[planetv1.ApplyRuleSetRequest],
) (*connect.Response[planetv1.ApplyRuleSetResponse], error) {
	err := h.useCase.Execute(ctx, apply_rule_set_usecase.In{
		Request: rules.RequestID(req.Msg.GetRequestId()),
		Set:     rules.SetName(req.Msg.GetRuleSet()),
		Source:  h.source,
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.ApplyRuleSetResponse{}), nil
	case errors.Is(err, rules.ErrUnknownSet), errors.Is(err, rules.ErrNoRequest), errors.Is(err, rules.ErrLongRequest):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
