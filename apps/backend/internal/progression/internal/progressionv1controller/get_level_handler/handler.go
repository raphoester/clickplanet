package get_level_handler

import (
	"context"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
)

type Query interface {
	Level(ctx context.Context, account accounts.AccountID) (*progressionv1.GetLevelResponse, error)
}

func New(query Query) GetLevelHandler {
	return GetLevelHandler{query: query}
}

type GetLevelHandler struct {
	query Query
}

func (h GetLevelHandler) GetLevel(
	ctx context.Context,
	_ *connect.Request[progressionv1.GetLevelRequest],
) (*connect.Response[progressionv1.GetLevelResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	level, err := h.query.Level(ctx, account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(level), nil
}
