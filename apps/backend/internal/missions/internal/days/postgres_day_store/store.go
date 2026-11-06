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
		SELECT day_id, season, number, starts_at, ends_at FROM mission_days WHERE starts_at <= $1 AND $1 < ends_at
	`, at.UTC()))
}

func (s *Store) DayBefore(ctx context.Context, day days.Day) (days.Day, error) {
	return dayOf(s.db.On(ctx).QueryRowContext(ctx, `
		SELECT day_id, season, number, starts_at, ends_at FROM mission_days WHERE season = $1 AND number = $2 - 1
	`, int64(day.Snapshot().Season), int64(day.Snapshot().Number)))
}

func dayOf(row *sql.Row) (days.Day, error) {
	var (
		snapshot       days.DaySnapshot
		id             string
		season, number int64
	)
	err := row.Scan(&id, &season, &number, &snapshot.StartsAt, &snapshot.EndsAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return days.Day{}, days.ErrNoDay
	case err != nil:
		return days.Day{}, fmt.Errorf("failed to read a day: %w", err)
	}
	snapshot.ID = days.DayID(id)
	snapshot.Season, snapshot.Number = uint32(season), uint32(number) //nolint:gosec // CHECK (season >= 0), CHECK (number > 0).
	return days.DayOf(snapshot)                                       //nolint:wrapcheck // it names the day.
}

func (s *Store) KeepDay(ctx context.Context, day days.Day) error {
	snapshot := day.Snapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO mission_days (day_id, season, number, starts_at, ends_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (day_id) DO UPDATE SET starts_at = excluded.starts_at, ends_at = excluded.ends_at
	`, string(snapshot.ID), int64(snapshot.Season), int64(snapshot.Number), snapshot.StartsAt, snapshot.EndsAt); err != nil {
		return fmt.Errorf("failed to keep the day %s: %w", snapshot.ID, err)
	}
	return nil
}
