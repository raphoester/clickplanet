package get_ledger_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler/changes_query"
)

type Query interface {
	Ledger(ctx context.Context, since, until time.Time) (*planetv1.GetLedgerResponse, error)
}

func New(query Query) GetLedgerHandler {
	return GetLedgerHandler{query: query}
}

type GetLedgerHandler struct {
	query Query
}

func (h GetLedgerHandler) GetLedger(
	ctx context.Context,
	req *connect.Request[planetv1.GetLedgerRequest],
) (*connect.Response[planetv1.GetLedgerResponse], error) {
	var since, until time.Time
	if at := req.Msg.GetSince(); at != nil {
		since = at.AsTime()
	}
	if at := req.Msg.GetUntil(); at != nil {
		until = at.AsTime()
	}

	answer, err := h.query.Ledger(ctx, since, until)
	if errors.Is(err, changes_query.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(answer), nil
}
