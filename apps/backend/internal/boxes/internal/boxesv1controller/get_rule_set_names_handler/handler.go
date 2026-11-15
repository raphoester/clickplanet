package get_rule_set_names_handler

import (
	"context"

	"connectrpc.com/connect"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
)

type Query interface {
	Names(ctx context.Context) ([]string, error)
}

func New(query Query) GetRuleSetNamesHandler {
	return GetRuleSetNamesHandler{query: query}
}

type GetRuleSetNamesHandler struct {
	query Query
}

func (h GetRuleSetNamesHandler) GetRuleSetNames(
	ctx context.Context,
	_ *connect.Request[boxesv1.GetRuleSetNamesRequest],
) (*connect.Response[boxesv1.GetRuleSetNamesResponse], error) {
	names, err := h.query.Names(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&boxesv1.GetRuleSetNamesResponse{Names: names}), nil
}
