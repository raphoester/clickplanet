package season_totals_query

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

type Planet interface {
	GetLedgerTotals(
		ctx context.Context,
		req *connect.Request[planetv1.GetLedgerTotalsRequest],
	) (*connect.Response[planetv1.GetLedgerTotalsResponse], error)
}

// Planet scans a season of its ledger: seconds, so the caller that leaves does not stop it.
const askTimeout = 2 * time.Minute

func New(seasons Seasons, planet Planet, clock cptime.Clock) *Query {
	return &Query{seasons: seasons, planet: planet, clock: clock, kept: map[season.Number]*planetv1.LedgerTotals{}}
}

type Query struct {
	seasons Seasons
	planet  Planet
	clock   cptime.Clock

	asking singleflight.Group
	mu     sync.Mutex
	kept   map[season.Number]*planetv1.LedgerTotals
}

func (q *Query) Totals(ctx context.Context, number season.Number) (*planetv1.LedgerTotals, error) {
	if totals, ok := q.keptTotals(number); ok {
		return totals, nil
	}
	calendar, err := q.seasons.Calendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	window, err := endedWindowOf(calendar, number, q.clock.Now())
	if err != nil {
		return nil, err
	}

	asked := q.asking.DoChan(strconv.FormatUint(uint64(number), 10), func() (any, error) {
		return q.ask(context.WithoutCancel(ctx), window)
	})
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("stopped waiting for season %d's totals: %w", number, ctx.Err())
	case answer := <-asked:
		if answer.Err != nil {
			return nil, answer.Err
		}
		return answer.Val.(*planetv1.LedgerTotals), nil //nolint:forcetypeassert // ask returns no other type.
	}
}

func (q *Query) ask(ctx context.Context, window season.Window) (*planetv1.LedgerTotals, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := q.planet.GetLedgerTotals(ctx, connect.NewRequest(&planetv1.GetLedgerTotalsRequest{
		Since: timestamppb.New(window.StartsAt), Until: timestamppb.New(window.EndsAt),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet for season %d's totals: %w", window.Number, err)
	}
	q.keep(window.Number, res.Msg.GetTotals())
	return res.Msg.GetTotals(), nil
}

func endedWindowOf(calendar season.Calendar, number season.Number, now time.Time) (season.Window, error) {
	for _, window := range calendar.EndedBy(now) {
		if window.Number == number {
			return window, nil
		}
	}
	return season.Window{}, fmt.Errorf("season %d: %w", number, season.ErrNotEnded)
}

func (q *Query) keptTotals(number season.Number) (*planetv1.LedgerTotals, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	totals, ok := q.kept[number]
	return totals, ok
}

func (q *Query) keep(number season.Number, totals *planetv1.LedgerTotals) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.kept[number] = totals
}
