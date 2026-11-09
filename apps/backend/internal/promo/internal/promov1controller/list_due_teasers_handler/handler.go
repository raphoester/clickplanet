package list_due_teasers_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

type Query interface {
	Due(ctx context.Context) ([]*promov1.Teaser, error)
}

func New(query Query) ListDueTeasersHandler {
	return ListDueTeasersHandler{query: query}
}

type ListDueTeasersHandler struct {
	query Query
}

// Unavailable when publishing could not say what was teased: the job tries at its next run.
func (h ListDueTeasersHandler) ListDueTeasers(
	ctx context.Context,
	_ *connect.Request[promov1.ListDueTeasersRequest],
) (*connect.Response[promov1.ListDueTeasersResponse], error) {
	due, err := h.query.Due(ctx)
	var busy *teasers.BusyError
	switch {
	case errors.As(err, &busy):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it, and passes a refusal of publishing's on.
	}
	return connect.NewResponse(&promov1.ListDueTeasersResponse{Teasers: due}), nil
}
