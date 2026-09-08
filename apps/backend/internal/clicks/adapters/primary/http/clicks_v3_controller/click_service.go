package clicks_v3_controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

// ClickService implements the generated service interface and nothing else.
// It never sees an http.ResponseWriter, which is the point of serving the
// contract with Connect rather than by hand.
type ClickService struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	logger              logging.Logger
}

var _ planetv1connect.ClickServiceHandler = (*ClickService)(nil)

func NewClickService(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	logger logging.Logger,
) *ClickService {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	return &ClickService{
		clickHandlerService: clickHandlerService,
		tilesChecker:        tilesChecker,
		logger:              logger,
	}
}

func (s *ClickService) Click(
	ctx context.Context,
	req *connect.Request[planetv1.ClickRequest],
) (*connect.Response[planetv1.ClickResponse], error) {
	err := s.clickHandlerService.HandleClick(ctx, req.Msg.GetTileId(), req.Msg.GetCountryId())
	if errors.Is(err, domain.ErrInvalidArgument) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		s.logger.Error("failed to handle click", lf.Err(err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}

	return connect.NewResponse(&planetv1.ClickResponse{}), nil
}

func (s *ClickService) MapDensity(
	_ context.Context,
	_ *connect.Request[planetv1.MapDensityRequest],
) (*connect.Response[planetv1.MapDensityResponse], error) {
	return connect.NewResponse(&planetv1.MapDensityResponse{
		Density: s.tilesChecker.MaxIndex(),
	}), nil
}
