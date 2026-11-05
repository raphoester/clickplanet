package season_replay_query

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

type Planet interface {
	GetMapHistory(
		ctx context.Context,
		req *connect.Request[planetv1.GetMapHistoryRequest],
	) (*connect.Response[planetv1.GetMapHistoryResponse], error)
}

var ErrNoFinale = errors.New("this season had no finale")

const (
	seasonStep = 30 * time.Minute
	finaleStep = 10 * time.Second
	// Planet reads its whole ledger: a minute or more, so the caller that leaves does not stop it.
	askTimeout = 10 * time.Minute
)

// Part is a season, or its finale alone.
type Part struct {
	Season season.Number
	Finale bool
}

func New(seasons Seasons, planet Planet, clock cptime.Clock) *Query {
	return &Query{seasons: seasons, planet: planet, clock: clock, kept: map[Part]*planetv1.MapHistory{}}
}

type Query struct {
	seasons Seasons
	planet  Planet
	clock   cptime.Clock

	asking singleflight.Group
	mu     sync.Mutex
	kept   map[Part]*planetv1.MapHistory
}

func (q *Query) Replay(ctx context.Context, part Part) (*planetv1.MapHistory, error) {
	if history, ok := q.keptHistory(part); ok {
		return history, nil
	}
	calendar, err := q.seasons.Calendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	window, err := calendar.Ended(part.Season, q.clock.Now())
	if err != nil {
		return nil, err //nolint:wrapcheck // the calendar names the season.
	}
	if part.Finale && !window.Finale.Planned() {
		return nil, fmt.Errorf("season %d: %w", part.Season, ErrNoFinale)
	}

	asked := q.asking.DoChan(fmt.Sprintf("%d/%t", part.Season, part.Finale), func() (any, error) {
		return q.ask(context.WithoutCancel(ctx), part, window)
	})
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("stopped waiting for season %d's replay: %w", part.Season, ctx.Err())
	case answer := <-asked:
		if answer.Err != nil {
			return nil, answer.Err
		}
		return answer.Val.(*planetv1.MapHistory), nil //nolint:forcetypeassert // ask returns no other type.
	}
}

func (q *Query) ask(ctx context.Context, part Part, window season.Window) (*planetv1.MapHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	since, step := window.StartsAt, seasonStep
	if part.Finale {
		since, step = window.FinaleStartsAt(), finaleStep
	}
	res, err := q.planet.GetMapHistory(ctx, connect.NewRequest(&planetv1.GetMapHistoryRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(window.EndsAt), Step: durationpb.New(step),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet for season %d's map history: %w", part.Season, err)
	}
	q.keep(part, res.Msg.GetHistory())
	return res.Msg.GetHistory(), nil
}

func (q *Query) keptHistory(part Part) (*planetv1.MapHistory, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	history, ok := q.kept[part]
	return history, ok
}

func (q *Query) keep(part Part, history *planetv1.MapHistory) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.kept[part] = history
}
