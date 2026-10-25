package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Sets interface {
	Sets(ctx context.Context) ([]*planetv1.RuleSet, error)
}

type Layers interface {
	Baselines(ctx context.Context) ([]*planetv1.RuleBaseline, error)
	Override(ctx context.Context) (*planetv1.RuleOverride, error)
}

func New(sets Sets, layers Layers) GetRulesHandler {
	return GetRulesHandler{sets: sets, layers: layers}
}

type GetRulesHandler struct {
	sets   Sets
	layers Layers
}

func (h GetRulesHandler) GetRules(
	ctx context.Context,
	_ *connect.Request[planetv1.GetRulesRequest],
) (*connect.Response[planetv1.GetRulesResponse], error) {
	sets, err := h.sets.Sets(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}
	baselines, err := h.layers.Baselines(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}
	override, err := h.layers.Override(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(&planetv1.GetRulesResponse{Sets: sets, Baselines: baselines, Override: override}), nil
}
