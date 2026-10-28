package get_seasons_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
)

type Query interface {
	Seasons(ctx context.Context) (*gamemasterv1.GetSeasonsResponse, error)
}

func New(query Query) GetSeasonsHandler {
	return GetSeasonsHandler{query: query}
}

type GetSeasonsHandler struct {
	query Query
}

func (h GetSeasonsHandler) GetSeasons(
	ctx context.Context,
	_ *connect.Request[gamemasterv1.GetSeasonsRequest],
) (*connect.Response[gamemasterv1.GetSeasonsResponse], error) {
	answer, err := h.query.Seasons(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(answer), nil
}
