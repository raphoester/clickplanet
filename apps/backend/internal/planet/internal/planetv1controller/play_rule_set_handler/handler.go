package play_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Rules interface {
	Play(ctx context.Context, choice rules.Choice) (rules.Snapshot, bool, error)
}

func New(rules Rules) PlayRuleSetHandler {
	return PlayRuleSetHandler{rules: rules}
}

type PlayRuleSetHandler struct {
	rules Rules
}

func (h PlayRuleSetHandler) PlayRuleSet(
	ctx context.Context,
	req *connect.Request[planetv1.PlayRuleSetRequest],
) (*connect.Response[planetv1.PlayRuleSetResponse], error) {
	_, _, err := h.rules.Play(ctx, rules.Choice{
		Set: rules.SetName(req.Msg.GetRuleSet()), DecidedAt: req.Msg.GetDecidedAt().AsTime(),
	})
	switch {
	case errors.Is(err, rules.ErrUnknownSet), errors.Is(err, rules.ErrInvalidSet):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.PlayRuleSetResponse{}), nil
}
