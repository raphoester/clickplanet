package postgres_entry_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ schedule.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

const columns = `id, rule_set, starts_at, ends_at, priority, made_at`

func (s *Store) Entry(ctx context.Context, id schedule.EntryID) (schedule.Entry, error) {
	entry, err := scan(s.db.On(ctx).QueryRowContext(ctx,
		`SELECT `+columns+` FROM schedule_entries WHERE id = $1`, uuid.UUID(id)))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return schedule.Entry{}, fmt.Errorf("%s: %w", id, schedule.ErrUnknownEntry)
	case err != nil:
		return schedule.Entry{}, fmt.Errorf("failed to read the entry %s: %w", id, err)
	}
	return entry, nil
}

func (s *Store) Unended(ctx context.Context, at time.Time) ([]schedule.Entry, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx,
		`SELECT `+columns+` FROM schedule_entries WHERE ends_at > $1`, at.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []schedule.Entry
	for rows.Next() {
		entry, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read an entry of the schedule: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the schedule: %w", err)
	}
	return entries, nil
}

func (s *Store) Save(ctx context.Context, entry schedule.Entry) error {
	kept := entry.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO schedule_entries (`+columns+`) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET rule_set = EXCLUDED.rule_set, starts_at = EXCLUDED.starts_at,
			ends_at = EXCLUDED.ends_at, priority = EXCLUDED.priority, made_at = EXCLUDED.made_at
	`, uuid.UUID(kept.ID), string(kept.Set), kept.StartsAt.UTC(), kept.EndsAt.UTC(), int64(kept.Priority), kept.MadeAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the entry %s: %w", kept.ID, err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id schedule.EntryID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM schedule_entries WHERE id = $1`, uuid.UUID(id)); err != nil {
		return fmt.Errorf("failed to delete the entry %s: %w", id, err)
	}
	return nil
}

func scan(row interface{ Scan(dest ...any) error }) (schedule.Entry, error) {
	var (
		id       uuid.UUID
		set      string
		priority int64
		kept     schedule.EntrySnapshot
	)
	if err := row.Scan(&id, &set, &kept.StartsAt, &kept.EndsAt, &priority, &kept.MadeAt); err != nil {
		return schedule.Entry{}, err //nolint:wrapcheck // each caller names what it read.
	}
	kept.ID, kept.Set, kept.Priority = schedule.EntryID(id), schedule.SetName(set), schedule.Priority(priority) //nolint:gosec // CHECK (priority >= 0 AND priority <= 4294967295).
	return kept.Restore(), nil
}
