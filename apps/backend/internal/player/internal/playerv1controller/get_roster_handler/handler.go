package get_roster_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

const maxAge = 5

type Query interface {
	Roster() *playerv1.GetRosterResponse
}

func New(query Query) GetRosterHandler {
	return GetRosterHandler{query: query}
}

type GetRosterHandler struct {
	query Query
}

func (h GetRosterHandler) GetRoster(
	_ context.Context,
	_ *connect.Request[playerv1.GetRosterRequest],
) (*connect.Response[playerv1.GetRosterResponse], error) {
	res := connect.NewResponse(h.query.Roster())
	res.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	return res, nil
}
