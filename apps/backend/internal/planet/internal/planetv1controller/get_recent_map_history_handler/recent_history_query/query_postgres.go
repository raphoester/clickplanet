package recent_history_query

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Tiles clamps each range to the map's last tile.
type Tiles interface {
	Stage() clicks.Stage
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
	Kept(ctx context.Context, kept clicks.Stage) (clicks.DenseBatch, error)
}

// Eras names the board the world plays, and the eras a window covers.
type Eras interface {
	Latest(ctx context.Context, played clicks.MapID) (eras.Era, error)
	Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]eras.Era, error)
}

type Resets interface {
	After(ctx context.Context, at time.Time) ([]resets.Reset, error)
}

type Forgettings interface {
	Forgetting(ctx context.Context) (forgetting.Forgetting, error)
}

// A build reads a day of the ledger: the caller that leaves does not stop it.
const buildTimeout = 2 * time.Minute

func NewPostgresQuery(
	db cppg.Conn, tiles Tiles, eras Eras, resets Resets, forgettings Forgettings, clock cptime.Clock,
) *PostgresQuery {
	return &PostgresQuery{db: db, tiles: tiles, eras: eras, resets: resets, forgettings: forgettings, clock: clock}
}

type PostgresQuery struct {
	db          cppg.Conn
	tiles       Tiles
	eras        Eras
	resets      Resets
	forgettings Forgettings
	clock       cptime.Clock

	building singleflight.Group
	mu       sync.Mutex
	kept     *built
}

type built struct {
	until     time.Time
	forgotten forgetting.Forgetting
	history   *planetv1.MapHistory
}

// RecentHistory is built once per step for every caller, and again after a revert, and answers when the next step's
// replaces it. What the reverts forgot is read before the build: one that lands during it builds again.
func (q *PostgresQuery) RecentHistory(ctx context.Context) (*planetv1.MapHistory, time.Time, error) {
	recent := ledger.RecentAt(q.clock.Now())
	forgotten, err := q.forgettings.Forgetting(ctx)
	if err != nil {
		return nil, time.Time{}, err //nolint:wrapcheck // it names what it read.
	}
	if history, ok := q.keptAt(recent.Until, forgotten); ok {
		return history, recent.Next(), nil
	}
	ready := q.building.DoChan(recent.Until.Format(time.RFC3339)+"/"+string(forgotten), func() (any, error) {
		return q.build(context.WithoutCancel(ctx), recent, forgotten)
	})
	select {
	case <-ctx.Done():
		return nil, time.Time{}, fmt.Errorf("stopped waiting for the map's recent history: %w", ctx.Err())
	case answer := <-ready:
		if answer.Err != nil {
			return nil, time.Time{}, answer.Err
		}
		return answer.Val.(*planetv1.MapHistory), recent.Next(), nil //nolint:forcetypeassert // build returns no other type.
	}
}

// The events from the head on keep their order by the primary key. The ones before the window are read for the rewind
// to see through what a revert forgot. Each names its map and its board, and the rewind keeps the world's on the board it
// plays.
const events = `
	SELECT map, board, kind, tile, country, previous, taken_at, payload, forgotten
	FROM ledger_events
	WHERE position >= $1
	ORDER BY position
`

func (q *PostgresQuery) build(
	ctx context.Context, recent ledger.Recent, forgotten forgetting.Forgetting,
) (*planetv1.MapHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, buildTimeout)
	defer cancel()

	after, err := q.resets.After(ctx, recent.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	for _, reset := range after {
		recent = recent.Wiped(reset.At)
	}
	world, err := q.eras.Latest(ctx, clicks.World)
	if err != nil {
		return nil, fmt.Errorf("failed to read the board the world plays: %w", err)
	}
	// The map before the switch to the board the world plays is on another board: no rewind reaches it.
	recent = recent.Wiped(world.StartedAt)
	if recent.Empty() {
		nothing := &planetv1.MapHistory{}
		q.keep(built{until: recent.Until, forgotten: forgotten, history: nothing})
		return nothing, nil
	}
	covering, err := q.eras.Covering(ctx, clicks.World, recent.Since, recent.Until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the world's eras: %w", err)
	}
	rewind, err := ledger.NewRewind(recent.Since, recent.Until, recent.Step, world.Stage())
	if err != nil {
		return nil, fmt.Errorf("failed to start the map's recent history: %w", err)
	}

	// The map before the ledger: a change made between the two reads is then in the ledger, and rewound.
	now, err := q.world(ctx, world.Stage())
	if err != nil {
		return nil, err
	}
	if err := q.replay(ctx, rewind); err != nil {
		return nil, err
	}
	history := planetmessage.MapHistory(rewind.Frames(now), ledger.InEra(len(covering)-1, rewind.Hits()), covering)
	q.keep(built{until: recent.Until, forgotten: forgotten, history: history})
	return history, nil
}

// The world's map as it plays, or as postgres keeps it on the board of its era while an arena's plays.
func (q *PostgresQuery) world(ctx context.Context, world clicks.Stage) (clicks.DenseBatch, error) {
	if q.tiles.Stage().Map != clicks.World {
		kept, err := q.tiles.Kept(ctx, world)
		if err != nil {
			return clicks.DenseBatch{}, fmt.Errorf("failed to read the world's map kept: %w", err)
		}
		return kept, nil
	}
	now, err := q.tiles.StateBatchDense(1, math.MaxUint32)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the map: %w", err)
	}
	return now, nil
}

func (q *PostgresQuery) replay(ctx context.Context, rewind *ledger.Rewind) error {
	var head int64
	if err := q.db.On(ctx).QueryRowContext(ctx, `SELECT COALESCE((SELECT head FROM ledger_head), 0)`).Scan(&head); err != nil {
		return fmt.Errorf("failed to read the ledger's head: %w", err)
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, events, head)
	if err != nil {
		return fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entry ledger.Entry
		if err := rows.Scan(&entry.Stage.Map, &entry.Stage.Board, &entry.Kind, &entry.Tile, &entry.Country, &entry.Previous,
			&entry.At, &entry.Payload, &entry.Forgotten); err != nil {
			return fmt.Errorf("failed to scan an event: %w", err)
		}
		entry.At = entry.At.UTC()
		if err := rewind.Record(entry); err != nil {
			return fmt.Errorf("failed to rewind an event: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the ledger: %w", err)
	}
	return nil
}

func (q *PostgresQuery) keptAt(until time.Time, forgotten forgetting.Forgetting) (*planetv1.MapHistory, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.kept == nil || !q.kept.until.Equal(until) || q.kept.forgotten != forgotten {
		return nil, false
	}
	return q.kept.history, true
}

func (q *PostgresQuery) keep(build built) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.kept == nil || !build.until.Before(q.kept.until) {
		q.kept = &build
	}
}
