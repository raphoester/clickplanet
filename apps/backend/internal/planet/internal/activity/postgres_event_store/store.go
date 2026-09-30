// Package postgres_event_store keeps the activity in activity.events, one row per event.
package postgres_event_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity/inmemory_event_buffer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db, chunk: defaultChunk}
}

type Store struct {
	db cppg.QuerierBeginner

	// chunk bounds one DELETE, so a prune cut by its timeout keeps what it did.
	chunk int
}

const defaultChunk = 10_000

var _ inmemory_event_buffer.Persistence = (*Store)(nil)

var columns = []string{
	"at", "kind", "scope", "account", "signed_in",
	"tile", "country", "outcome", "held", "map_start", "map_end", "off_map", "delay_us",
}

// Save is one COPY, however long postgres was away.
func (s *Store) Save(ctx context.Context, events []activity.Event) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save the activity: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("events", columns...))
	if err != nil {
		return fmt.Errorf("failed to start copying events: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, event := range events {
		if _, err := stmt.ExecContext(ctx, row(event)...); err != nil {
			return fmt.Errorf("failed to copy an event: %w", err)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to copy events: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the activity: %w", err)
	}

	return nil
}

// row is an event in the order of columns, with NULL in every column its kind does not have.
func row(event activity.Event) []any {
	var tile, country, outcome, held, start, end, offMap, delay any

	switch event.Kind {
	case activity.KindClick:
		tile, country, outcome = int64(event.Tile), event.Country, string(event.Outcome)
	case activity.KindTake:
		tile, country, held = int64(event.Tile), event.Country, nullIfEmpty(event.Held)
	case activity.KindMap:
		outcome, start, end, offMap = string(event.Outcome), int64(event.Start), int64(event.End), event.OffMap
	case activity.KindBoxCaught:
		delay = event.Delay.Microseconds()
	}

	return []any{
		event.At, string(event.Kind), event.Caller.Scope, account(event.Caller.Account), event.Caller.SignedIn,
		tile, country, outcome, held, start, end, offMap, delay,
	}
}

func account(id cpsession.AccountID) any {
	if id == cpsession.NoAccount {
		return nil
	}

	return uuid.UUID(id).String()
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.deleteChunks(ctx, `DELETE FROM events WHERE id IN (SELECT id FROM events WHERE at < $1 LIMIT $2)`, cutoff)
}

// DeleteOldestBeyond goes by id; an id a failed flush burned makes it keep a few less than kept.
func (s *Store) DeleteOldestBeyond(ctx context.Context, kept int) (int64, error) {
	var newest int64
	if err := s.db.QueryRowContext(ctx, `SELECT coalesce(max(id), 0) FROM events`).Scan(&newest); err != nil {
		return 0, fmt.Errorf("failed to read the newest event: %w", err)
	}

	last := newest - int64(kept)
	if last <= 0 {
		return 0, nil
	}

	return s.deleteChunks(ctx, `DELETE FROM events WHERE id IN (SELECT id FROM events WHERE id <= $1 LIMIT $2)`, last)
}

func (s *Store) deleteChunks(ctx context.Context, query string, bound any) (int64, error) {
	var total int64
	for {
		result, err := s.db.ExecContext(ctx, query, bound, s.chunk)
		if err != nil {
			return total, fmt.Errorf("failed to delete events: %w", err)
		}

		deleted, err := result.RowsAffected()
		if err != nil {
			return total, fmt.Errorf("failed to count the events deleted: %w", err)
		}

		total += deleted
		if deleted < int64(s.chunk) {
			return total, nil
		}
	}
}
