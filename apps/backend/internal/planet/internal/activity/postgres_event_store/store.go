// Package postgres_event_store keeps the activity in activity.events, one row per event.
package postgres_event_store

import (
	"context"
	"database/sql"
	"encoding/json"
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

var columns = []string{"at", "kind", "scope", "account", "signed_in", "data"}

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
		data, err := dataOf(event)
		if err != nil {
			return fmt.Errorf("failed to encode a %s event: %w", event.Kind, err)
		}

		if _, err := stmt.ExecContext(ctx, event.At, string(event.Kind), event.Caller.Scope,
			account(event.Caller.Account), event.Caller.SignedIn, data); err != nil {
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

func account(id cpsession.AccountID) any {
	if id == cpsession.NoAccount {
		return nil
	}

	return uuid.UUID(id).String()
}

// The data column of each kind. A kind with none stores NULL.
type (
	clickData struct {
		Tile    uint32 `json:"tile"`
		Country string `json:"country"`
		Outcome string `json:"outcome"`
	}

	// takeData leaves held out for a tile nobody held.
	takeData struct {
		Tile    uint32 `json:"tile"`
		Country string `json:"country"`
		Held    string `json:"held,omitempty"`
		Cleared bool   `json:"cleared,omitempty"`
	}

	mapData struct {
		Start   uint32 `json:"start"`
		End     uint32 `json:"end"`
		OffMap  bool   `json:"off_map"`
		Outcome string `json:"outcome"`
	}

	caughtData struct {
		DelayUS int64 `json:"delay_us"`
	}
)

func payloadOf(event activity.Event) (any, bool) {
	switch event.Kind {
	case activity.KindClick:
		return clickData{Tile: event.Tile, Country: event.Country, Outcome: string(event.Outcome)}, true
	case activity.KindTake:
		return takeData{Tile: event.Tile, Country: event.Country, Held: event.Held, Cleared: event.Cleared}, true
	case activity.KindMap:
		return mapData{Start: event.Start, End: event.End, OffMap: event.OffMap, Outcome: string(event.Outcome)}, true
	case activity.KindBoxCaught:
		return caughtData{DelayUS: event.Delay.Microseconds()}, true
	default:
		return nil, false
	}
}

// dataOf is a string, not bytes: lib/pq copies bytes as bytea, which a jsonb column refuses.
func dataOf(event activity.Event) (sql.Null[string], error) {
	payload, ok := payloadOf(event)
	if !ok {
		return sql.Null[string]{}, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return sql.Null[string]{}, fmt.Errorf("failed to encode the data: %w", err)
	}

	return sql.Null[string]{V: string(encoded), Valid: true}, nil
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
