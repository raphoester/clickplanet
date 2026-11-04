package get_titles_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
)

type Query interface {
	Titles(ctx context.Context, account accounts.AccountID) (*progressionv1.GetTitlesResponse, error)
}

func New(query Query) GetTitlesHandler {
	return GetTitlesHandler{query: query}
}

type GetTitlesHandler struct {
	query Query
}

func (h GetTitlesHandler) GetTitles(
	ctx context.Context,
	_ *connect.Request[progressionv1.GetTitlesRequest],
) (*connect.Response[progressionv1.GetTitlesResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	dashboard, err := h.query.Titles(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(dashboard), nil
}
