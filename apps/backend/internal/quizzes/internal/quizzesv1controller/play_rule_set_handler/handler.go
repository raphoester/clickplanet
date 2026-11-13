package play_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

type Rules interface {
	Play(ctx context.Context, choice cprules.Choice) (cprules.Snapshot[*quizzesv1.Rules], bool, error)
}

func New(rules Rules) PlayRuleSetHandler {
	return PlayRuleSetHandler{rules: rules}
}

type PlayRuleSetHandler struct {
	rules Rules
}

func (h PlayRuleSetHandler) PlayRuleSet(
	ctx context.Context,
	req *connect.Request[quizzesv1.PlayRuleSetRequest],
) (*connect.Response[quizzesv1.PlayRuleSetResponse], error) {
	_, _, err := h.rules.Play(ctx, cprules.Choice{
		Set: cprules.SetName(req.Msg.GetRuleSet()), DecidedAt: req.Msg.GetDecidedAt().AsTime(),
	})
	switch {
	case errors.Is(err, cprules.ErrUnknownSet), errors.Is(err, cprules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&quizzesv1.PlayRuleSetResponse{}), nil
}
