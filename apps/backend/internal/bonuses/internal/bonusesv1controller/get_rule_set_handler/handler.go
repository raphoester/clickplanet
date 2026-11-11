package get_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler/rule_set_query"
)

type Query interface {
	Played(ctx context.Context, name string) (*bonusesv1.Rules, error)
}

func New(query Query) GetRuleSetHandler {
	return GetRuleSetHandler{query: query}
}

type GetRuleSetHandler struct {
	query Query
}

func (h GetRuleSetHandler) GetRuleSet(
	ctx context.Context,
	req *connect.Request[bonusesv1.GetRuleSetRequest],
) (*connect.Response[bonusesv1.GetRuleSetResponse], error) {
	played, err := h.query.Played(ctx, req.Msg.GetName())
	switch {
	case errors.Is(err, rule_set_query.ErrNoSet):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&bonusesv1.GetRuleSetResponse{Played: played}), nil
}
