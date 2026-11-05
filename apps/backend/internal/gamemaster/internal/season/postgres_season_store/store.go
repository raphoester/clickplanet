package postgres_season_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
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

const read = `
	SELECT s.number, (EXTRACT(EPOCH FROM s.break_before) * 1000000)::bigint, (EXTRACT(EPOCH FROM s.length) * 1000000)::bigint,
		(EXTRACT(EPOCH FROM f.length) * 1000000)::bigint, f.rule_set, f.description, f.times
	FROM seasons AS s LEFT JOIN season_finales AS f ON f.season = s.number`

func (s *Store) Calendar(ctx context.Context) (season.Calendar, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, read)
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
	finale := snapshot.Finale
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		WITH kept AS (
			INSERT INTO seasons (number, break_before, length)
			VALUES ($1, $2 * interval '1 microsecond', $3 * interval '1 microsecond')
			ON CONFLICT (number) DO UPDATE SET break_before = EXCLUDED.break_before, length = EXCLUDED.length
		), none AS (
			DELETE FROM season_finales WHERE season = $1 AND NOT $4::boolean
		)
		INSERT INTO season_finales (season, length, rule_set, description, times)
		SELECT $1, $5 * interval '1 microsecond', $6, $7, $8 WHERE $4::boolean
		ON CONFLICT (season) DO UPDATE SET length = EXCLUDED.length, rule_set = EXCLUDED.rule_set,
			description = EXCLUDED.description, times = EXCLUDED.times
	`, int64(snapshot.Number), snapshot.Break.Microseconds(), snapshot.Length.Microseconds(), finale.Planned(),
		finale.Length.Microseconds(), string(finale.Set), string(finale.Description), int64(finale.Times)); err != nil {
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
	var (
		number, brk, length       int64
		finaleLength, times       sql.NullInt64
		finaleSet, finaleDescribe sql.NullString
	)
	if err := row.Scan(&number, &brk, &length, &finaleLength, &finaleSet, &finaleDescribe, &times); err != nil {
		return season.Season{}, err //nolint:wrapcheck // each caller names what it read.
	}
	snapshot := season.Snapshot{
		Number: season.Number(number), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
		Break:  time.Duration(brk) * time.Microsecond, Length: time.Duration(length) * time.Microsecond,
	}
	if finaleLength.Valid {
		snapshot.Finale = season.Finale{
			Length: time.Duration(finaleLength.Int64) * time.Microsecond, Set: schedule.SetName(finaleSet.String),
			Description: schedule.Description(finaleDescribe.String),
			Times:       schedule.Times(times.Int64), //nolint:gosec // CHECK (times BETWEEN 1 AND 10).
		}
	}
	return snapshot.Restore(), nil
}
