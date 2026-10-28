package postgres_season_store

import (
	"context"
	"fmt"
	"time"

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

const columns = `number, (EXTRACT(EPOCH FROM break_before) * 1000000)::bigint, (EXTRACT(EPOCH FROM length) * 1000000)::bigint`

func (s *Store) Calendar(ctx context.Context) (season.Calendar, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT `+columns+` FROM seasons`)
	if err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var seasons []season.Season
	for rows.Next() {
		kept, err := scan(rows)
		if err != nil {
			return season.Calendar{}, fmt.Errorf("failed to read a season: %w", err)
		}
		seasons = append(seasons, kept)
	}
	if err := rows.Err(); err != nil {
		return season.Calendar{}, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return season.NewCalendar(seasons...), nil
}

func (s *Store) Save(ctx context.Context, kept season.Season) error {
	snapshot := kept.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO seasons (number, break_before, length)
		VALUES ($1, $2 * interval '1 microsecond', $3 * interval '1 microsecond')
		ON CONFLICT (number) DO UPDATE SET break_before = EXCLUDED.break_before, length = EXCLUDED.length
	`, int64(snapshot.Number), snapshot.Break.Microseconds(), snapshot.Length.Microseconds()); err != nil {
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

func scan(row interface{ Scan(dest ...any) error }) (season.Season, error) {
	var number, brk, length int64
	if err := row.Scan(&number, &brk, &length); err != nil {
		return season.Season{}, err //nolint:wrapcheck // each caller names what it read.
	}
	return season.Snapshot{
		Number: season.Number(number), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
		Break:  time.Duration(brk) * time.Microsecond, Length: time.Duration(length) * time.Microsecond,
	}.Restore(), nil
}
