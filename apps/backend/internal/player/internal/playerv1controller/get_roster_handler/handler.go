// Package get_roster_handler serves player.v1.PlayerService/GetRoster.
package get_roster_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
)

// maxAge lets a proxy serve one roster to every client that asks within it. The roster changes slowly, and
// every client polls it.
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
	roster := h.useCase.Execute()

	entries := make([]*playerv1.RosterEntry, 0, len(roster))
	for _, entry := range roster {
		entries = append(entries, &playerv1.RosterEntry{
			Name:      entry.Name,
			Tag:       string(entry.Tag),
			CountryId: entry.Country,
			Guest:     entry.Guest,
			Admin:     entry.Admin,
		})
	}

	res := connect.NewResponse(&playerv1.GetRosterResponse{Entries: entries})
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
