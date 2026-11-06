package get_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
)

type Query interface {
	Rules(ctx context.Context) (*missionsv1.GetRulesResponse, error)
}

func New(query Query) GetRulesHandler {
	return GetRulesHandler{query: query}
}

type GetRulesHandler struct {
	query Query
}

func (h GetRulesHandler) GetRules(
	ctx context.Context,
	_ *connect.Request[missionsv1.GetRulesRequest],
) (*connect.Response[missionsv1.GetRulesResponse], error) {
	answer, err := h.query.Rules(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	res := connect.NewResponse(answer)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
