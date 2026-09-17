// Package get_roster_handler serves player.v1.PlayerService/GetRoster.
package get_roster_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

// maxAge lets a proxy serve one roster to every client that asks within it, for clients that still poll it.
const maxAge = 5

type UseCase interface {
	Execute() []presence.Entry
}

func New(useCase UseCase) GetRosterHandler {
	return GetRosterHandler{useCase: useCase}
}

type GetRosterHandler struct {
	useCase UseCase
}

// GetRoster needs no token: the session interceptor does not list it.
func (h GetRosterHandler) GetRoster(
	_ context.Context,
	_ *connect.Request[playerv1.GetRosterRequest],
) (*connect.Response[playerv1.GetRosterResponse], error) {
	res := connect.NewResponse(&playerv1.GetRosterResponse{Entries: playermessage.RosterEntries(h.useCase.Execute())})
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
