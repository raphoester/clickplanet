package count_arrivals_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
)

type Query interface {
	Counts(ctx context.Context, since, until time.Time) (*authv1.CountArrivalsResponse, error)
}

func New(query Query) CountArrivalsHandler {
	return CountArrivalsHandler{query: query}
}

type CountArrivalsHandler struct {
	query Query
}

func (h CountArrivalsHandler) CountArrivals(
	ctx context.Context,
	req *connect.Request[authv1.CountArrivalsRequest],
) (*connect.Response[authv1.CountArrivalsResponse], error) {
	var since, until time.Time
	if asked := req.Msg.GetSince(); asked != nil {
		since = asked.AsTime()
	}
	if asked := req.Msg.GetUntil(); asked != nil {
		until = asked.AsTime()
	}

	counted, err := h.query.Counts(ctx, since, until)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(counted), nil
}
