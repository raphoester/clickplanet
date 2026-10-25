package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Sets interface {
	Sets() []rules.Set
}

type Changes interface {
	Changes(ctx context.Context) ([]*planetv1.RuleChange, error)
}

func New(sets Sets, changes Changes) GetRulesHandler {
	return GetRulesHandler{sets: sets, changes: changes}
}

type GetRulesHandler struct {
	sets    Sets
	changes Changes
}

func (h GetRulesHandler) GetRules(
	ctx context.Context,
	_ *connect.Request[planetv1.GetRulesRequest],
) (*connect.Response[planetv1.GetRulesResponse], error) {
	changes, err := h.changes.Changes(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	sets := h.sets.Sets()
	encoded := make([]*planetv1.RuleSet, 0, len(sets))
	for _, set := range sets {
		encoded = append(encoded, &planetv1.RuleSet{
			Name:  string(set.Name),
			Tempo: set.Tempo,
			Boxes: &planetv1.BoxPace{
				MinInterval:       durationpb.New(set.Boxes.MinInterval),
				MaxInterval:       durationpb.New(set.Boxes.MaxInterval),
				MaxChargesPerHour: uint32(max(set.Boxes.MaxChargesPerHour, 0)), //nolint:gosec // capped below.
			},
		})
	}

	return connect.NewResponse(&planetv1.GetRulesResponse{Sets: encoded, Changes: changes}), nil
}
