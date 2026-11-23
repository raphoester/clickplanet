package world_replay

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Eras names the eras of the world a window covers.
type Eras interface {
	Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]eras.Era, error)
}

// Carries reads the carry-over of a switch of the world's board, from the board before it to the board after it.
type Carries interface {
	Carry(from, to eras.Board) (ledger.Carry, error)
}

func NewPostgresReplay(db cppg.Conn, eras Eras, carries Carries) *PostgresReplay {
	return &PostgresReplay{db: db, eras: eras, carries: carries}
}

// PostgresReplay reads the world's ledger from the blank map before a time: the last reset, or the first event.
type PostgresReplay struct {
	db      cppg.Conn
	eras    Eras
	carries Carries
}

// Plan is the world's eras from the blank map before a time to the end of a window, oldest first, each later one with
// its carry-over, and the resets from the blank map on.
type Plan struct {
	Eras   []eras.Era
	blank  time.Time
	resets []time.Time
	cuts   []cut
}

// A cut is an era's start, with the carry-over from the board of the era before it.
type cut struct {
	era   eras.Era
	carry ledger.Carry
}

func (q *PostgresReplay) Plan(ctx context.Context, since, until time.Time) (Plan, error) {
	resets, err := q.resets(ctx)
	if err != nil {
		return Plan{}, err
	}
	resets, blank := blankBefore(resets, since)
	all, err := q.eras.Covering(ctx, clicks.World, blank, until)
	if err != nil {
		return Plan{}, fmt.Errorf("failed to read the world's eras: %w", err)
	}
	if len(all) == 0 {
		return Plan{}, fmt.Errorf("failed to read the world's eras: %w", eras.ErrNoEra)
	}
	cuts, err := q.cutsOf(all)
	if err != nil {
		return Plan{}, err
	}
	return Plan{Eras: all, blank: blank, resets: resets, cuts: cuts}, nil
}

// Positions follow time, so a replay stops at the first event its reader is done with. Each event names its map and
// its board, and the reader keeps the world's on the board of the era in force at its time.
const events = `
	SELECT position, map, board, kind, tile, COALESCE(scope, ''), COALESCE(account::text, ''), country, previous,
		taken_at, payload, forgotten
	FROM ledger_events
	WHERE taken_at >= $1
	ORDER BY position
`

// Replay plays the plan's events, resets and cuts up to until on replayed, on the stage of its first era, and answers
// the position after the last event it played.
func (q *PostgresReplay) Replay(ctx context.Context, plan Plan, until time.Time, replayed ledger.Replayed) (ledger.Position, error) {
	read, resets, cuts, err := q.play(ctx, replayed, plan.resets, plan.cuts, events, plan.blank)
	if err != nil {
		return 0, err
	}
	changesUpTo(replayed, resets, cuts, until)
	return read, nil
}

// play answers the position after the last event it played, and the resets and cuts after that event.
func (q *PostgresReplay) play(
	ctx context.Context, replayed ledger.Replayed, resets []time.Time, cuts []cut, query string, args ...any,
) (ledger.Position, []time.Time, []cut, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	read := ledger.Position(0)
	for rows.Next() {
		var (
			position int64
			entry    ledger.Entry
		)
		if err := rows.Scan(&position, &entry.Stage.Map, &entry.Stage.Board, &entry.Kind, &entry.Tile, &entry.Scope,
			&entry.Account, &entry.Country, &entry.Previous, &entry.At, &entry.Payload, &entry.Forgotten); err != nil {
			return 0, nil, nil, fmt.Errorf("failed to scan an event: %w", err)
		}
		entry.At = entry.At.UTC()
		if replayed.Done(entry) {
			break
		}
		resets, cuts = changesUpTo(replayed, resets, cuts, entry.At)
		if err := replayed.Record(entry); err != nil {
			return 0, nil, nil, fmt.Errorf("failed to replay an event: %w", err)
		}
		read = ledger.Position(position) + 1 //nolint:gosec // CHECK (position >= 0).
	}
	if err := rows.Err(); err != nil {
		return 0, nil, nil, fmt.Errorf("failed to read the ledger: %w", err)
	}
	return read, resets, cuts, nil
}

func (q *PostgresReplay) cutsOf(all []eras.Era) ([]cut, error) {
	cuts := make([]cut, 0, len(all)-1)
	for i, era := range all[1:] {
		carry, err := q.carries.Carry(all[i].Board, era.Board)
		if err != nil {
			return nil, fmt.Errorf("failed to read the carry-over from %s to %s: %w", all[i].Board.Name, era.Board.Name, err)
		}
		cuts = append(cuts, cut{era: era, carry: carry})
	}
	return cuts, nil
}

// ToNow replays the world's map from its last blank map to now, for a revert.
type ToNow struct {
	replay *PostgresReplay
	clock  cptime.Clock
}

func NewToNow(replay *PostgresReplay, clock cptime.Clock) ToNow {
	return ToNow{replay: replay, clock: clock}
}

// Replay plays the world's ledger on what start makes of the stage of its first era.
func (t ToNow) Replay(ctx context.Context, start func(first clicks.Stage) ledger.Replayed) (ledger.Position, error) {
	now := t.clock.Now()
	plan, err := t.replay.Plan(ctx, now, now)
	if err != nil {
		return 0, err
	}
	return t.replay.Replay(ctx, plan, now, start(plan.Eras[0].Stage()))
}

// changesUpTo plays the resets and the cuts at or before at, in time order, and returns those after it.
func changesUpTo(replayed ledger.Replayed, resets []time.Time, cuts []cut, at time.Time) ([]time.Time, []cut) {
	for {
		reset := len(resets) > 0 && !resets[0].After(at)
		cutting := len(cuts) > 0 && !cuts[0].era.StartedAt.After(at)
		switch {
		case reset && (!cutting || !cuts[0].era.StartedAt.Before(resets[0])):
			replayed.Wipe(resets[0])
			resets = resets[1:]
		case cutting:
			replayed.Cut(cuts[0].era.StartedAt, cuts[0].era.Board.ID, cuts[0].carry)
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
func (q *PostgresReplay) resets(ctx context.Context) ([]time.Time, error) {
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
