package get_rule_set_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_handler/rule_set_query"
)

type Query interface {
	Played(ctx context.Context, name string) (*planetv1.Rules, error)
}

func New(query Query) GetRuleSetHandler {
	return GetRuleSetHandler{query: query}
}

type GetRuleSetHandler struct {
	query Query
}

func (h GetRuleSetHandler) GetRuleSet(
	ctx context.Context,
	req *connect.Request[planetv1.GetRuleSetRequest],
) (*connect.Response[planetv1.GetRuleSetResponse], error) {
	played, err := h.query.Played(ctx, req.Msg.GetName())
	switch {
	case errors.Is(err, rule_set_query.ErrNoSet):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.GetRuleSetResponse{Played: played}), nil
}
