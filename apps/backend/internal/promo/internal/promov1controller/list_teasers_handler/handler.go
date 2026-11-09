package list_teasers_handler

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
)

// A teaser goes out a day or so before its battle: a week is far more than any job asks.
const maxWithin = 7 * 24 * time.Hour

type Query interface {
	Teasers(ctx context.Context, within time.Duration) ([]*promov1.Teaser, error)
}

func New(query Query) ListTeasersHandler {
	return ListTeasersHandler{query: query}
}

type ListTeasersHandler struct {
	query Query
}

func (h ListTeasersHandler) ListTeasers(
	ctx context.Context,
	req *connect.Request[promov1.ListTeasersRequest],
) (*connect.Response[promov1.ListTeasersResponse], error) {
	within := req.Msg.GetWithin().AsDuration()
	if within <= 0 || within > maxWithin {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("the window is %s, and must be more than nothing and %s at most", within, maxWithin))
	}
	teasers, err := h.query.Teasers(ctx, within)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&promov1.ListTeasersResponse{Teasers: teasers}), nil
}
