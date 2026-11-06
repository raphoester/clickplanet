package list_renames_handler

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/list_renames_handler/renames_query"
)

type Query interface {
	Renames(ctx context.Context, filter renames_query.Filter) (*playerv1.ListRenamesResponse, error)
}

func New(query Query) ListRenamesHandler {
	return ListRenamesHandler{query: query}
}

type ListRenamesHandler struct {
	query Query
}

func (h ListRenamesHandler) ListRenames(
	ctx context.Context,
	req *connect.Request[playerv1.ListRenamesRequest],
) (*connect.Response[playerv1.ListRenamesResponse], error) {
	filter := renames_query.Filter{Limit: req.Msg.GetLimit(), Since: time.Unix(0, 0)}
	if id := req.Msg.GetAccountId(); id != "" {
		// Read as everyone, it would answer other accounts' renames to an operator reading one account's.
		account, err := players.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		filter.Account = account
	}
	if since := req.Msg.GetSince(); since != nil {
		filter.Since = since.AsTime()
	}

	renames, err := h.query.Renames(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to read the renames: %w", err)
	}

	res := connect.NewResponse(renames)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
