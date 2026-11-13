package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type Sets interface {
	Sets(ctx context.Context) ([]*quizzesv1.RuleSet, error)
}

type Playing interface {
	Name() cprules.SetName
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
	_ *connect.Request[quizzesv1.GetRulesRequest],
) (*connect.Response[quizzesv1.GetRulesResponse], error) {
	sets, err := h.sets.Sets(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the query says what failed.
	}

	return connect.NewResponse(&quizzesv1.GetRulesResponse{Sets: sets, Playing: string(h.playing.Name())}), nil
}
