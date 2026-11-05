package get_players_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Query interface {
	Players(ctx context.Context, accounts []players.AccountID) (*playerv1.GetPlayersResponse, error)
}

func New(query Query) GetPlayersHandler {
	return GetPlayersHandler{query: query}
}

type GetPlayersHandler struct {
	query Query
}

func (h GetPlayersHandler) GetPlayers(
	ctx context.Context,
	req *connect.Request[playerv1.GetPlayersRequest],
) (*connect.Response[playerv1.GetPlayersResponse], error) {
	asked := make([]players.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		account, err := players.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	found, err := h.query.Players(ctx, asked)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(found)
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
