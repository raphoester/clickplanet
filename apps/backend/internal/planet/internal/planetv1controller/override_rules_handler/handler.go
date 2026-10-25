package override_rules_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in override_rules_usecase.In) error
}

var ErrNoEnd = errors.New("an override has an end")

func New(useCase UseCase) OverrideRulesHandler {
	return OverrideRulesHandler{useCase: useCase}
}

type OverrideRulesHandler struct {
	useCase UseCase
}

func (h OverrideRulesHandler) OverrideRules(
	ctx context.Context,
	req *connect.Request[planetv1.OverrideRulesRequest],
) (*connect.Response[planetv1.OverrideRulesResponse], error) {
	if req.Msg.GetUntil() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoEnd)
	}

	err := h.useCase.Execute(ctx, override_rules_usecase.In{
		Set:   rules.SetName(req.Msg.GetRuleSet()),
		Until: req.Msg.GetUntil().AsTime(),
	})
	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.OverrideRulesResponse{}), nil
	case errors.Is(err, rules.ErrUnknownSet), errors.Is(err, rules.ErrInvalidOverride):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
