package get_nameplates_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// MaxAsked is how many accounts one call may name: a page of names, never the whole game.
const MaxAsked = 100

var ErrTooMany = errors.New("too many accounts asked at once")

// Query is GetPlayers' own, so a name is drawn the same wherever the page reads it.
type Query interface {
	Players(ctx context.Context, accounts []players.AccountID) (*playerv1.GetPlayersResponse, error)
}

func New(query Query) GetNameplatesHandler {
	return GetNameplatesHandler{query: query}
}

type GetNameplatesHandler struct {
	query Query
}

func (h GetNameplatesHandler) GetNameplates(
	ctx context.Context,
	req *connect.Request[playerv1.GetNameplatesRequest],
) (*connect.Response[playerv1.GetNameplatesResponse], error) {
	ids := req.Msg.GetAccountIds()
	if len(ids) > MaxAsked {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: %d, at most %d", ErrTooMany, len(ids), MaxAsked))
	}
	asked := make([]players.AccountID, 0, len(ids))
	for _, id := range ids {
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

	answer := &playerv1.GetNameplatesResponse{Nameplates: make([]*playerv1.Nameplate, 0, len(found.GetPlayers()))}
	for _, player := range found.GetPlayers() {
		answer.Nameplates = append(answer.Nameplates, &playerv1.Nameplate{
			AccountId: player.GetAccountId(),
			Name:      player.GetName(),
			Color:     player.GetColor(),
			Admin:     player.GetAdmin(),
			Guest:     player.GetGuest(),
		})
	}

	res := connect.NewResponse(answer)
	// The page keeps what it read: an answer kept on the way could undo a rename it then learns of.
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
