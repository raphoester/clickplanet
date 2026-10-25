package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Sets interface {
	Sets(ctx context.Context) ([]*planetv1.RuleSet, error)
}

func New(sets Sets) GetRulesHandler {
	return GetRulesHandler{sets: sets}
}

type GetRulesHandler struct {
	sets Sets
}

func (h GetRulesHandler) GetRules(
	ctx context.Context,
	_ *connect.Request[planetv1.GetRulesRequest],
) (*connect.Response[planetv1.GetRulesResponse], error) {
	sets, err := h.sets.Sets(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(&planetv1.GetRulesResponse{Sets: sets}), nil
}
