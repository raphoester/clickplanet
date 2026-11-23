package history_query

import (
	"context"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/world_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

// World reads the world's ledger from the blank map before a time, each era on its board.
type World interface {
	Plan(ctx context.Context, since, until time.Time) (world_replay.Plan, error)
	Replay(ctx context.Context, plan world_replay.Plan, until time.Time, replayed ledger.Replayed) (ledger.Position, error)
}

func NewPostgresQuery(world World) *PostgresQuery {
	return &PostgresQuery{world: world}
}

type PostgresQuery struct {
	world World
}

// History draws each era of the world the window covers on its board, with the cut: the frames before an era's start
// are on the board before, the map is carried over at it, and the first frame after it holds every tile held on its
// own. The map at the window's start is every take before it, from the last reset at or before it.
func (q *PostgresQuery) History(ctx context.Context, since, until time.Time, step time.Duration) (*planetv1.MapHistory, error) {
	plan, err := q.world.Plan(ctx, since, until)
	if err != nil {
		return nil, fmt.Errorf("failed to plan the world's replay: %w", err)
	}
	history, err := ledger.NewHistory(since, until, step, plan.Eras[0].Stage())
	if err != nil {
		return nil, fmt.Errorf("failed to start the map's history: %w", err)
	}
	if _, err := q.world.Replay(ctx, plan, until, history); err != nil {
		return nil, fmt.Errorf("failed to replay the world: %w", err)
	}
	return planetmessage.MapHistory(history.Frames(), history.Hits(), eras.From(plan.Eras, since)), nil
}
