package planetv1controller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
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
	Peek(key string) ratelimit.State
}

type ClickService struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	mapReader           DenseMapReader
	subscriber          UpdatesSubscriber
	heartbeat           time.Duration
	budgets             ClickBudgetReader
	bonuses             BonusRegistry
	booster             ClickBooster
	clock               xtime.Provider
}

var _ planetv1connect.ClickServiceHandler = (*ClickService)(nil)

// NewClickService takes a nil budgets reader for a server that does not rate
// limit clicks; its answers then carry no allowance, and a client shows none.
//
// A nil bonuses registry is a server with boxes switched off: no offer ever
// reaches a stream and ClaimBonus answers Unimplemented, which is the same
// shape as chat being disabled.
func NewClickService(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	mapReader DenseMapReader,
	subscriber UpdatesSubscriber,
	heartbeat time.Duration,
	budgets ClickBudgetReader,
	bonuses BonusRegistry,
	booster ClickBooster,
	clock xtime.Provider,
) *ClickService {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	if clock == nil {
		clock = xtime.ActualProvider{}
	}

	return &ClickService{
		clickHandlerService: clickHandlerService,
		tilesChecker:        tilesChecker,
		mapReader:           mapReader,
		subscriber:          subscriber,
		heartbeat:           heartbeat,
		budgets:             budgets,
		bonuses:             bonuses,
		booster:             booster,
		clock:               clock,
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
	if state, limited := ctxutil.GetClickBudget(ctx); limited {
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
		res.Budget = EncodeBudget(s.budgets.Peek(connectutil.RateLimitKey(ctx)))
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

	// The bonus feed is a second subscription on the same connection, not a
	// second stream: a PlanetEvent oneof is exactly what lets one connection
	// carry a kind of event that did not exist when the client was written.
	//
	// It is also the only feed on this stream that is *addressed* — an offer
	// arrives here because this caller was drawn for it, and reaches no other
	// connection. That is why the registry is keyed on the same scope the
	// throttle is: a caller is one entrant however many tabs it has open.
	var bonuses <-chan bonus.Event
	if s.bonuses != nil {
		events, leave := s.bonuses.Attend(connectutil.RateLimitKey(ctx))
		defer leave()
		bonuses = events
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

		case event, open := <-bonuses:
			if !open {
				return nil
			}

			frame := bonusEvent(event)
			if frame == nil {
				continue
			}

			if err := stream.Send(frame); err != nil {
				return err
			}
		}
	}
}

func cacheControl() string {
	return fmt.Sprintf("public, max-age=%d", mapMaxAge)
}
