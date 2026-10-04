package get_history_handler

import (
	"context"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Query interface {
	History(ctx context.Context, viewer messages.AccountID) (*chatv1.GetHistoryResponse, error)
}

func New(query Query) GetHistoryHandler {
	return GetHistoryHandler{query: query}
}

type GetHistoryHandler struct {
	query Query
}

func (h GetHistoryHandler) GetHistory(
	ctx context.Context,
	_ *connect.Request[chatv1.GetHistoryRequest],
) (*connect.Response[chatv1.GetHistoryResponse], error) {
	history, err := h.query.History(ctx, messages.AccountIDOf(cpctx.GetAccount(ctx)))
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	res := connect.NewResponse(history)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
