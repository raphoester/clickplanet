package get_player_by_name_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_player_by_name_handler/holder_query"
)

const maxAge = 10

var ErrNoName = errors.New("name a player")

type Query interface {
	Holder(ctx context.Context, folded string) (*playerv1.GetPlayerByNameResponse, error)
}

func New(query Query) GetPlayerByNameHandler {
	return GetPlayerByNameHandler{query: query}
}

type GetPlayerByNameHandler struct {
	query Query
}

func (h GetPlayerByNameHandler) GetPlayerByName(
	ctx context.Context,
	req *connect.Request[playerv1.GetPlayerByNameRequest],
) (*connect.Response[playerv1.GetPlayerByNameResponse], error) {
	folded := players.FoldedOf(req.Msg.GetName())
	if folded == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoName)
	}

	holder, err := h.query.Holder(ctx, folded)
	switch {
	case errors.Is(err, holder_query.ErrNoHolder):
		return nil, connect.NewError(connect.CodeNotFound, holder_query.ErrNoHolder)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	res := connect.NewResponse(holder)
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
