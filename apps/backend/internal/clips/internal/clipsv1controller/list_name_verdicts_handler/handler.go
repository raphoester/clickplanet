package list_name_verdicts_handler

import (
	"context"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
)

type Query interface {
	Verdicts(ctx context.Context, hiddenOnly bool) (*clipsv1.ListNameVerdictsResponse, error)
}

func New(query Query) ListNameVerdictsHandler {
	return ListNameVerdictsHandler{query: query}
}

type ListNameVerdictsHandler struct {
	query Query
}

func (h ListNameVerdictsHandler) ListNameVerdicts(
	ctx context.Context,
	req *connect.Request[clipsv1.ListNameVerdictsRequest],
) (*connect.Response[clipsv1.ListNameVerdictsResponse], error) {
	listed, err := h.query.Verdicts(ctx, req.Msg.GetHiddenOnly())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(listed), nil
}
