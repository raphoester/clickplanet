package list_my_submissions_handler

import (
	"context"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Query interface {
	Mine(ctx context.Context, account submissions.AccountID) (*flagsv1.ListMySubmissionsResponse, error)
}

func New(query Query) ListMySubmissionsHandler {
	return ListMySubmissionsHandler{query: query}
}

type ListMySubmissionsHandler struct {
	query Query
}

func (h ListMySubmissionsHandler) ListMySubmissions(
	ctx context.Context,
	_ *connect.Request[flagsv1.ListMySubmissionsRequest],
) (*connect.Response[flagsv1.ListMySubmissionsResponse], error) {
	account, err := submissions.FounderOf(cpctx.GetAccount(ctx), cpctx.GetLinked(ctx))
	if err != nil {
		return connect.NewResponse(&flagsv1.ListMySubmissionsResponse{}), nil //nolint:nilerr // a guest founded none.
	}
	mine, err := h.query.Mine(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(mine), nil
}
