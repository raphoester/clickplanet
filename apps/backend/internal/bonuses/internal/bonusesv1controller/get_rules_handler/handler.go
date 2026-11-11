package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
)

type Sets interface {
	Sets(ctx context.Context) ([]*bonusesv1.RuleSet, error)
}

type Playing interface {
	Name() rules.SetName
}

func New(sets Sets, playing Playing) GetRulesHandler {
	return GetRulesHandler{sets: sets, playing: playing}
}

type GetRulesHandler struct {
	sets    Sets
	playing Playing
}

func (h GetRulesHandler) GetRules(
	ctx context.Context,
	_ *connect.Request[bonusesv1.GetRulesRequest],
) (*connect.Response[bonusesv1.GetRulesResponse], error) {
	sets, err := h.sets.Sets(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(&bonusesv1.GetRulesResponse{Sets: sets, Playing: string(h.playing.Name())}), nil
}
