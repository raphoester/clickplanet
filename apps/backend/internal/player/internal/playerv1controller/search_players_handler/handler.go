package search_players_handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

const (
	defaultLimit = 20
	maxLimit     = 500
)

var ErrNoName = errors.New("type a part of a name")

type Query interface {
	Search(ctx context.Context, folded string, limit int) (*playerv1.SearchPlayersResponse, error)
}

func New(query Query) SearchPlayersHandler {
	return SearchPlayersHandler{query: query}
}

type SearchPlayersHandler struct {
	query Query
}

func (h SearchPlayersHandler) SearchPlayers(
	ctx context.Context,
	req *connect.Request[playerv1.SearchPlayersRequest],
) (*connect.Response[playerv1.SearchPlayersResponse], error) {
	folded := players.FoldedOf(strings.TrimSpace(req.Msg.GetName()))
	if folded == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoName)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultLimit
	}

	matches, err := h.query.Search(ctx, folded, min(limit, maxLimit))
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(matches)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
