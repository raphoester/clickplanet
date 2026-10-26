package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

type Sets interface {
	Sets(ctx context.Context) ([]*antibotv1.RuleSet, error)
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
	_ *connect.Request[antibotv1.GetRulesRequest],
) (*connect.Response[antibotv1.GetRulesResponse], error) {
	sets, err := h.sets.Sets(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(&antibotv1.GetRulesResponse{Sets: sets, Playing: string(h.playing.Name())}), nil
}
