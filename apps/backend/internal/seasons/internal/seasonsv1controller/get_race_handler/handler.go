package get_race_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
)

// The read the season stream sends from, made once: a reader that follows no stream, the clips job, asks it here.
type Query interface {
	Race(ctx context.Context) (*seasonsv1.Race, error)
}

func New(query Query) GetRaceHandler {
	return GetRaceHandler{query: query}
}

type GetRaceHandler struct {
	query Query
}

func (h GetRaceHandler) GetRace(
	ctx context.Context,
	_ *connect.Request[seasonsv1.GetRaceRequest],
) (*connect.Response[seasonsv1.GetRaceResponse], error) {
	race, err := h.query.Race(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(&seasonsv1.GetRaceResponse{Race: race}), nil
}
