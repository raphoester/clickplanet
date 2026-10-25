package get_hall_of_fame_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
)

type Query interface {
	HallOfFame(ctx context.Context) (*seasonsv1.GetHallOfFameResponse, error)
}

func New(query Query) GetHallOfFameHandler {
	return GetHallOfFameHandler{query: query}
}

type GetHallOfFameHandler struct {
	query Query
}

func (h GetHallOfFameHandler) GetHallOfFame(
	ctx context.Context,
	_ *connect.Request[seasonsv1.GetHallOfFameRequest],
) (*connect.Response[seasonsv1.GetHallOfFameResponse], error) {
	hall, err := h.query.HallOfFame(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(hall), nil
}
