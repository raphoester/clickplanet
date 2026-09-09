package planetv1controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
)

// mapMaxAge is short because the websocket carries everything newer.
const mapMaxAge = 5

// DenseMapReader reads a tile range without repeating an id per tile.
type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (domain.DenseBatch, error)
}

// ClickService implements the generated service interface and nothing else.
type ClickService struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	mapReader           DenseMapReader
}

var _ planetv1connect.ClickServiceHandler = (*ClickService)(nil)

func NewClickService(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	mapReader DenseMapReader,
) *ClickService {
	return &ClickService{
		clickHandlerService: clickHandlerService,
		tilesChecker:        tilesChecker,
		mapReader:           mapReader,
	}
}

func (s *ClickService) Click(
	ctx context.Context,
	req *connect.Request[planetv1.ClickRequest],
) (*connect.Response[planetv1.ClickResponse], error) {
	if err := s.clickHandlerService.HandleClick(ctx, req.Msg.GetTileId(), req.Msg.GetCountryId()); err != nil {
		return nil, err
	}

	return connect.NewResponse(&planetv1.ClickResponse{}), nil
}

func (s *ClickService) MapDensity(
	_ context.Context,
	_ *connect.Request[planetv1.MapDensityRequest],
) (*connect.Response[planetv1.MapDensityResponse], error) {
	res := connect.NewResponse(&planetv1.MapDensityResponse{
		Density: s.tilesChecker.MaxIndex(),
	})
	res.Header().Set("Cache-Control", cacheControl())

	return res, nil
}

func (s *ClickService) GetMap(
	_ context.Context,
	req *connect.Request[planetv1.GetMapRequest],
) (*connect.Response[planetv1.GetMapResponse], error) {
	end := req.Msg.GetEndTileId()
	if end == 0 {
		end = s.tilesChecker.MaxIndex()
	}

	batch, err := s.mapReader.StateBatchDense(req.Msg.GetStartTileId(), end)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidArgument, err)
	}

	res := connect.NewResponse(&planetv1.GetMapResponse{
		StartTileId: batch.Start,
		Codes:       batch.Codes,
		Tiles:       batch.Tiles,
	})
	res.Header().Set("Cache-Control", cacheControl())

	return res, nil
}

func cacheControl() string {
	return fmt.Sprintf("public, max-age=%d", mapMaxAge)
}
