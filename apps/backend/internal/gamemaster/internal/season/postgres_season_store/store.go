package postgres_season_store

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ season.Store = (*Store)(nil)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Calendar(ctx context.Context) (season.Calendar, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT number, starts_at, ends_at FROM seasons`)
	if err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var seasons []season.Season
	for rows.Next() {
		var (
			number int64
			kept   season.Snapshot
		)
		if err := rows.Scan(&number, &kept.StartsAt, &kept.EndsAt); err != nil {
			return season.Calendar{}, fmt.Errorf("failed to read a season: %w", err)
		}
		kept.Number = season.Number(number) //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
		seasons = append(seasons, kept.Restore())
	}
	if err := rows.Err(); err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return season.NewCalendar(seasons...), nil
}

func (s *Store) Save(ctx context.Context, kept season.Season) error {
	snapshot := kept.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO seasons (number, starts_at, ends_at) VALUES ($1, $2, $3)
		ON CONFLICT (number) DO UPDATE SET starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at
	`, int64(snapshot.Number), snapshot.StartsAt.UTC(), snapshot.EndsAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep season %d: %w", snapshot.Number, err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, number season.Number) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM seasons WHERE number = $1`, int64(number)); err != nil {
		return fmt.Errorf("failed to delete season %d: %w", number, err)
	}
	return nil
}
