package planetv1controller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

const mapMaxAge = 5

type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (domain.DenseBatch, error)
}

type UpdatesSubscriber interface {
	Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error)
}

// ClickBudgetReader reports a caller's click allowance without spending it,
// under the same key the rate limiter spends it under.
type ClickBudgetReader interface {
	Peek(key string) cpratelimit.State
}

type ClickService struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	mapReader           DenseMapReader
	subscriber          UpdatesSubscriber
	heartbeat           time.Duration
	budgets             ClickBudgetReader
}

var _ planetv1connect.ClickServiceHandler = (*ClickService)(nil)

// NewClickService takes a nil budgets reader for a server that does not rate
// limit clicks; its answers then carry no allowance, and a client shows none.
func NewClickService(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	mapReader DenseMapReader,
	subscriber UpdatesSubscriber,
	heartbeat time.Duration,
	budgets ClickBudgetReader,
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
		budgets:             budgets,
	}
}

// Click answers with what the rate limiter had left after letting this click
// through. The reading is the interceptor's — taken at the moment the token was
// spent — so it is never a token behind what the server will enforce next.
func (s *ClickService) Click(
	ctx context.Context,
	req *connect.Request[planetv1.ClickRequest],
) (*connect.Response[planetv1.ClickResponse], error) {
	if err := s.clickHandlerService.HandleClick(ctx, req.Msg.GetTileId(), req.Msg.GetCountryId()); err != nil {
		return nil, err
	}

	res := &planetv1.ClickResponse{}
	if state, limited := cpctx.GetRateBudget(ctx); limited {
		res.Budget = EncodeBudget(state)
	}

	return connect.NewResponse(res), nil
}

// GetBudget is how a client that has just loaded learns its allowance. Every
// click answers with a fresh one afterwards, so this is asked once.
func (s *ClickService) GetBudget(
	ctx context.Context,
	_ *connect.Request[planetv1.GetBudgetRequest],
) (*connect.Response[planetv1.GetBudgetResponse], error) {
	res := &planetv1.GetBudgetResponse{}
	if s.budgets != nil {
		res.Budget = EncodeBudget(s.budgets.Peek(cpconnect.RateLimitKey(ctx)))
	}

	return connect.NewResponse(res), nil
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
		return nil, fmt.Errorf("%w: %w", domain.ErrInvalidArgument, err)
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
