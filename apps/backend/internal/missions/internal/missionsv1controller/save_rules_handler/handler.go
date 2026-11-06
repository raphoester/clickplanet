package save_rules_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
)

type UseCase interface {
	Execute(ctx context.Context, in *missionsv1.Rules) error
}

func New(useCase UseCase) SaveRulesHandler {
	return SaveRulesHandler{useCase: useCase}
}

type SaveRulesHandler struct {
	useCase UseCase
}

func (h SaveRulesHandler) SaveRules(
	ctx context.Context,
	req *connect.Request[missionsv1.SaveRulesRequest],
) (*connect.Response[missionsv1.SaveRulesResponse], error) {
	err := h.useCase.Execute(ctx, req.Msg.GetRules())
	switch {
	case err == nil:
		return connect.NewResponse(&missionsv1.SaveRulesResponse{}), nil
	case errors.Is(err, rules.ErrInvalidRules):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
}
