package get_country_sizes_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Borders interface {
	CountrySizes() []clicks.CountrySize
}

func New(borders Borders) GetCountrySizesHandler {
	return GetCountrySizesHandler{borders: borders}
}

type GetCountrySizesHandler struct {
	borders Borders
}

func (h GetCountrySizesHandler) GetCountrySizes(
	_ context.Context,
	_ *connect.Request[planetv1.GetCountrySizesRequest],
) (*connect.Response[planetv1.GetCountrySizesResponse], error) {
	sizes := h.borders.CountrySizes()

	answer := &planetv1.GetCountrySizesResponse{CountrySizes: make([]*planetv1.CountrySize, 0, len(sizes))}
	for _, size := range sizes {
		answer.CountrySizes = append(answer.CountrySizes, &planetv1.CountrySize{CountryId: size.Country, Tiles: size.Tiles})
	}

	return connect.NewResponse(answer), nil
}
