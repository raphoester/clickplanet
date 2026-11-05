package get_progress_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler/progress_query"
)

const maxAge = 10

type Query interface {
	Progress(ctx context.Context, account accounts.AccountID) (*progressionv1.GetProgressResponse, error)
}

func New(query Query) GetProgressHandler {
	return GetProgressHandler{query: query}
}

type GetProgressHandler struct {
	query Query
}

func (h GetProgressHandler) GetProgress(
	ctx context.Context,
	req *connect.Request[progressionv1.GetProgressRequest],
) (*connect.Response[progressionv1.GetProgressResponse], error) {
	account, err := accounts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	progress, err := h.query.Progress(ctx, account)
	switch {
	case errors.Is(err, progress_query.ErrNotShown):
		return nil, connect.NewError(connect.CodeNotFound, progress_query.ErrNotShown)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(progress)
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
