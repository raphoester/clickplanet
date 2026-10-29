package postgres_day_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ days.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) DayAt(ctx context.Context, at time.Time) (days.Day, error) {
	return dayOf(s.db.On(ctx).QueryRowContext(ctx, `
		SELECT day_id, starts_at, ends_at FROM mission_days WHERE starts_at <= $1 AND $1 < ends_at
	`, at.UTC()))
}

func (s *Store) DayBefore(ctx context.Context, day days.Day) (days.Day, error) {
	return dayOf(s.db.On(ctx).QueryRowContext(ctx, `
		SELECT day_id, starts_at, ends_at FROM mission_days WHERE ends_at = $1
	`, day.Snapshot().StartsAt))
}

func dayOf(row *sql.Row) (days.Day, error) {
	var snapshot days.DaySnapshot
	var id string
	err := row.Scan(&id, &snapshot.StartsAt, &snapshot.EndsAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return days.Day{}, days.ErrNoDay
	case err != nil:
		return days.Day{}, fmt.Errorf("failed to read a day: %w", err)
	}
	snapshot.ID = days.DayID(id)
	return days.DayOf(snapshot) //nolint:wrapcheck // it names the day.
}

func (s *Store) KeepDay(ctx context.Context, day days.Day) error {
	snapshot := day.Snapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO mission_days (day_id, starts_at, ends_at) VALUES ($1, $2, $3)
		ON CONFLICT (day_id) DO UPDATE SET starts_at = excluded.starts_at, ends_at = excluded.ends_at
	`, string(snapshot.ID), snapshot.StartsAt, snapshot.EndsAt); err != nil {
		return fmt.Errorf("failed to keep the day %s: %w", snapshot.ID, err)
	}
	return nil
}
