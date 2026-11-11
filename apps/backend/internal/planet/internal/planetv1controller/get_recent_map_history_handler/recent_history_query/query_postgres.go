package recent_history_query

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Tiles clamps each range to the map's last tile.
type Tiles interface {
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
	Kept(ctx context.Context, kept clicks.MapID) (clicks.DenseBatch, error)
}

type Arenas interface {
	Stretches(ctx context.Context) ([]arenas.Stretch, error)
}

type Resets interface {
	After(ctx context.Context, at time.Time) ([]resets.Reset, error)
}

// A build reads a day of the ledger: the caller that leaves does not stop it.
const buildTimeout = 2 * time.Minute

func NewPostgresQuery(db cppg.Conn, tiles Tiles, arenas Arenas, resets Resets, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, tiles: tiles, arenas: arenas, resets: resets, clock: clock}
}

type PostgresQuery struct {
	db     cppg.Conn
	tiles  Tiles
	arenas Arenas
	resets Resets
	clock  cptime.Clock

	building singleflight.Group
	mu       sync.Mutex
	kept     *built
}

type built struct {
	until   time.Time
	history *planetv1.MapHistory
}

// RecentHistory is built once per step for every caller, and answers when the next step's replaces it.
func (q *PostgresQuery) RecentHistory(ctx context.Context) (*planetv1.MapHistory, time.Time, error) {
	recent := ledger.RecentAt(q.clock.Now())
	if history, ok := q.keptAt(recent.Until); ok {
		return history, recent.Next(), nil
	}
	ready := q.building.DoChan(recent.Until.Format(time.RFC3339), func() (any, error) {
		return q.build(context.WithoutCancel(ctx), recent)
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

// The events from the head on keep their order by the primary key; taken_at has no index.
const events = `
	SELECT kind, tile, COALESCE(account::text, ''), country, previous, taken_at, payload
	FROM ledger_events
	WHERE position >= $1 AND taken_at >= $2 AND NOT forgotten
	ORDER BY position
`

func (q *PostgresQuery) build(ctx context.Context, recent ledger.Recent) (*planetv1.MapHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, buildTimeout)
	defer cancel()

	after, err := q.resets.After(ctx, recent.Since)
	if err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	for _, reset := range after {
		recent = recent.Wiped(reset.At)
	}
	if recent.Empty() {
		nothing := &planetv1.MapHistory{}
		q.keep(recent.Until, nothing)
		return nothing, nil
	}
	stretches, err := q.arenas.Stretches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the arenas: %w", err)
	}
	maps := arenas.MapsOf(stretches)
	rewind, err := ledger.NewRewind(recent.Since, recent.Until, recent.Step, maps.World())
	if err != nil {
		return nil, fmt.Errorf("failed to start the map's recent history: %w", err)
	}

	// The map before the ledger: a change made between the two reads is then in the ledger, and rewound.
	now, err := q.world(ctx, maps)
	if err != nil {
		return nil, err
	}
	if err := q.replay(ctx, recent.Since, rewind); err != nil {
		return nil, err
	}
	history := planetmessage.MapHistory(rewind.Frames(now), rewind.Hits())
	q.keep(recent.Until, history)
	return history, nil
}

func (q *PostgresQuery) world(ctx context.Context, maps arenas.Maps) (clicks.DenseBatch, error) {
	if _, open := maps.Open(); open {
		kept, err := q.tiles.Kept(ctx, clicks.World)
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

func (q *PostgresQuery) replay(ctx context.Context, since time.Time, rewind *ledger.Rewind) error {
	var head int64
	if err := q.db.On(ctx).QueryRowContext(ctx, `SELECT COALESCE((SELECT head FROM ledger_head), 0)`).Scan(&head); err != nil {
		return fmt.Errorf("failed to read the ledger's head: %w", err)
	}
	rows, err := q.db.On(ctx).QueryContext(ctx, events, head, since)
	if err != nil {
		return fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entry ledger.Entry
		if err := rows.Scan(&entry.Kind, &entry.Tile, &entry.Account, &entry.Country, &entry.Previous, &entry.At,
			&entry.Payload); err != nil {
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

func (q *PostgresQuery) keptAt(until time.Time) (*planetv1.MapHistory, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.kept == nil || !q.kept.until.Equal(until) {
		return nil, false
	}
	return q.kept.history, true
}

func (q *PostgresQuery) keep(until time.Time, history *planetv1.MapHistory) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.kept == nil || until.After(q.kept.until) {
		q.kept = &built{until: until, history: history}
	}
}
