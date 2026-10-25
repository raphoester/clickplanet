package get_grounds_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Borders interface {
	Grounds() []clicks.Ground
}

func New(borders Borders) GetGroundsHandler {
	return GetGroundsHandler{borders: borders}
}

type GetGroundsHandler struct {
	borders Borders
}

func (h GetGroundsHandler) GetGrounds(
	_ context.Context,
	_ *connect.Request[planetv1.GetGroundsRequest],
) (*connect.Response[planetv1.GetGroundsResponse], error) {
	grounds := h.borders.Grounds()

	answer := &planetv1.GetGroundsResponse{Grounds: make([]*planetv1.Ground, 0, len(grounds))}
	for _, ground := range grounds {
		answer.Grounds = append(answer.Grounds, &planetv1.Ground{CountryId: ground.Country, Tiles: ground.Tiles})
	}

	return connect.NewResponse(answer), nil
}
