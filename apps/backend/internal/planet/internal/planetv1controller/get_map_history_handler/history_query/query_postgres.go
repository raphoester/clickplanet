package history_query

import (
	"context"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Eras names the eras of the world a window covers.
type Eras interface {
	Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]eras.Era, error)
}

func NewPostgresQuery(db cppg.Conn, eras Eras) *PostgresQuery {
	return &PostgresQuery{db: db, eras: eras}
}

type PostgresQuery struct {
	db   cppg.Conn
	eras Eras
}

// The map at the window's start is every take before it, from the last reset at or before it, where the map was blank.
// Positions follow time, so the read stops at the first event at or after the window's end. Each event names its map
// and its board, and the history keeps the world's on the board of the era in force at its time.
const events = `
	SELECT map, board, kind, tile, COALESCE(scope, ''), COALESCE(account::text, ''), country, previous, taken_at, payload
	FROM ledger_events
	WHERE taken_at >= $1
	ORDER BY position
`

// History draws each era of the world the window covers on its board, with the cut: the frames before an era's start
// are on the board before, and the first after it holds every tile held on its own.
func (q *PostgresQuery) History(ctx context.Context, since, until time.Time, step time.Duration) (*planetv1.MapHistory, error) {
	resets, err := q.resets(ctx)
	if err != nil {
		return nil, err
	}
	resets, blank := blankBefore(resets, since)
	// The eras from the blank map on: the one in force then is the board the first entries replay on.
	all, err := q.eras.Covering(ctx, clicks.World, blank, until)
	if err != nil {
		return nil, fmt.Errorf("failed to read the world's eras: %w", err)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("failed to read the world's eras: %w", eras.ErrNoEra)
	}
	history, err := ledger.NewHistory(since, until, step, all[0].Stage())
	if err != nil {
		return nil, fmt.Errorf("failed to start the map's history: %w", err)
	}
	cuts := all[1:]
	rows, err := q.db.On(ctx).QueryContext(ctx, events, blank)
	if err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var entry ledger.Entry
		if err := rows.Scan(&entry.Stage.Map, &entry.Stage.Board, &entry.Kind, &entry.Tile, &entry.Scope, &entry.Account,
			&entry.Country, &entry.Previous, &entry.At, &entry.Payload); err != nil {
			return nil, fmt.Errorf("failed to scan an event: %w", err)
		}
		entry.At = entry.At.UTC()
		if history.Done(entry) {
			break
		}
		resets, cuts = changesUpTo(history, resets, cuts, entry.At)
		if err := history.Record(entry); err != nil {
			return nil, fmt.Errorf("failed to replay an event: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	changesUpTo(history, resets, cuts, until)
	return planetmessage.MapHistory(history.Frames(), history.Hits(), eras.From(all, since)), nil
}

// changesUpTo plays the resets and the cuts at or before at on the history, in time order, and returns those after it.
func changesUpTo(history *ledger.History, resets []time.Time, cuts []eras.Era, at time.Time) ([]time.Time, []eras.Era) {
	for {
		reset := len(resets) > 0 && !resets[0].After(at)
		cut := len(cuts) > 0 && !cuts[0].StartedAt.After(at)
		switch {
		case reset && (!cut || !cuts[0].StartedAt.Before(resets[0])):
			history.Wipe(resets[0])
			resets = resets[1:]
		case cut:
			history.Cut(cuts[0].StartedAt, cuts[0].Board.ID)
			cuts = cuts[1:]
		default:
			return resets, cuts
		}
	}
}

// The resets from the last one at or before since, and that one's time; the zero time when none precedes since.
func blankBefore(resets []time.Time, since time.Time) ([]time.Time, time.Time) {
	last := -1
	for i, reset := range resets {
		if !reset.After(since) {
			last = i
		}
	}
	if last < 0 {
		return resets, time.Time{}
	}
	return resets[last:], resets[last]
}

// The map resets leave no event in the ledger.
func (q *PostgresQuery) resets(ctx context.Context) ([]time.Time, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT at FROM map_resets ORDER BY at`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var resets []time.Time
	for rows.Next() {
		var at time.Time
		if err := rows.Scan(&at); err != nil {
			return nil, fmt.Errorf("failed to read a reset: %w", err)
		}
		resets = append(resets, at.UTC())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	return resets, nil
}
