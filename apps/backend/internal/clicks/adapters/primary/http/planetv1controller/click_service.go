package planetv1controller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
)

const mapMaxAge = 5

type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (domain.DenseBatch, error)
}

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error)
}

type ClickService struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	mapReader           DenseMapReader
	subscriber          UpdatesSubscriber
	heartbeat           time.Duration
}

var _ planetv1connect.ClickServiceHandler = (*ClickService)(nil)

func NewClickService(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	mapReader DenseMapReader,
	subscriber UpdatesSubscriber,
	heartbeat time.Duration,
) *ClickService {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}

	return &ClickService{
		clickHandlerService: clickHandlerService,
		tilesChecker:        tilesChecker,
		mapReader:           mapReader,
		subscriber:          subscriber,
		heartbeat:           heartbeat,
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

// The request context is what unsubscribes, and it is cancelled however the stream ends.
func (s *ClickService) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[planetv1.ListenForEventsRequest],
	stream *connect.ServerStream[planetv1.PlanetEvent],
) error {
	updates, err := s.subscriber.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tile updates: %w", err)
	}

	heartbeat := time.NewTicker(s.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-heartbeat.C:
			if err := stream.Send(heartbeatEvent()); err != nil {
				return err
			}

		case update, open := <-updates:
			if !open {
				return nil
			}

			if err := stream.Send(tileUpdateEvent(update)); err != nil {
				return err
			}
		}
	}
}

func cacheControl() string {
	return fmt.Sprintf("public, max-age=%d", mapMaxAge)
}
