package admin_get_progress_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Query interface {
	Progress(ctx context.Context, account accounts.AccountID) (*progressionv1.AdminGetProgressResponse, error)
}

func New(query Query) AdminGetProgressHandler {
	return AdminGetProgressHandler{query: query}
}

type AdminGetProgressHandler struct {
	query Query
}

func (h AdminGetProgressHandler) GetProgress(
	ctx context.Context,
	req *connect.Request[progressionv1.AdminGetProgressRequest],
) (*connect.Response[progressionv1.AdminGetProgressResponse], error) {
	account, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	progress, err := h.query.Progress(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(progress)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
