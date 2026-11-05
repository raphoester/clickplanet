package get_ledger_totals_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Query interface {
	Totals(ctx context.Context, since, until time.Time) (*planetv1.LedgerTotals, error)
}

var ErrInvalidWindow = errors.New("the window must start before it ends")

func New(query Query) GetLedgerTotalsHandler {
	return GetLedgerTotalsHandler{query: query}
}

type GetLedgerTotalsHandler struct {
	query Query
}

func (h GetLedgerTotalsHandler) GetLedgerTotals(
	ctx context.Context,
	req *connect.Request[planetv1.GetLedgerTotalsRequest],
) (*connect.Response[planetv1.GetLedgerTotalsResponse], error) {
	since, until := req.Msg.GetSince().AsTime(), req.Msg.GetUntil().AsTime()
	if req.Msg.GetSince() == nil || req.Msg.GetUntil() == nil || !since.Before(until) {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidWindow)
	}
	totals, err := h.query.Totals(ctx, since, until)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&planetv1.GetLedgerTotalsResponse{Totals: totals}), nil
}
