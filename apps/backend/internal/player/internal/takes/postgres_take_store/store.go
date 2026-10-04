package postgres_take_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ takes.Store = (*Store)(nil)

func (s *Store) Position(ctx context.Context) (takes.Position, error) {
	var position int64
	err := s.db.QueryRowContext(ctx, `SELECT position FROM stats_position`).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, takes.ErrNotStarted
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read where the stats are counted to: %w", err)
	}
	return takes.Position(position), nil //nolint:gosec // CHECK (position >= start), start >= 0.
}

func (s *Store) Begin(ctx context.Context, start takes.Position) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting the takes: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	begun, err := tx.ExecContext(ctx,
		`INSERT INTO stats_position (start, position) VALUES ($1, $1) ON CONFLICT DO NOTHING`, bigint(start))
	if err != nil {
		return fmt.Errorf("failed to save where the stats start: %w", err)
	}
	rows, err := begun.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the stats had started: %w", err)
	}
	if rows == 0 {
		return takes.ErrStarted
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stats_baseline (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
		SELECT account_id, tiles_taken, streak_current, streak_best, streak_last_day FROM stats
		WHERE tiles_taken > 0
	`); err != nil {
		return fmt.Errorf("failed to copy the stats into the baseline: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the start: %w", err)
	}
	return nil
}

func (s *Store) Count(ctx context.Context, batch takes.Batch) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin counting a batch: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	position, err := lockedPosition(ctx, tx, `position`)
	if err != nil {
		return err
	}
	if position != batch.From() {
		return fmt.Errorf("%w: the batch is from %d, the stats are at %d", takes.ErrMoved, batch.From(), position)
	}

	current, err := statsOf(ctx, tx, batch.Accounts())
	if err != nil {
		return err
	}
	if err := saveTakes(ctx, tx, batch.Tallied(current)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE stats_position SET position = $1`, bigint(batch.Next())); err != nil {
		return fmt.Errorf("failed to save where the stats are counted to: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit a batch: %w", err)
	}
	return nil
}

func (s *Store) Rewind(ctx context.Context) (_ takes.Position, err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to begin a rewind: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	start, err := lockedPosition(ctx, tx, `start`)
	if err != nil {
		return 0, err
	}

	for _, statement := range []string{
		`UPDATE stats SET tiles_taken = 0, streak_current = 0, streak_best = 0, streak_last_day = NULL
		 WHERE tiles_taken > 0 AND account_id NOT IN (SELECT account_id FROM stats_baseline)`,
		`UPDATE stats SET
			tiles_taken = stats_baseline.tiles_taken,
			streak_current = stats_baseline.streak_current,
			streak_best = stats_baseline.streak_best,
			streak_last_day = stats_baseline.streak_last_day
		 FROM stats_baseline WHERE stats_baseline.account_id = stats.account_id`,
		`UPDATE stats_position SET position = start`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return 0, fmt.Errorf("failed to put the stats back to their start: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit a rewind: %w", err)
	}
	return start, nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM stats_baseline WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's baseline: %w", err)
	}
	return nil
}

func lockedPosition(ctx context.Context, tx *sql.Tx, column string) (takes.Position, error) {
	var position int64
	err := tx.QueryRowContext(ctx, `SELECT `+column+` FROM stats_position FOR UPDATE`).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, takes.ErrNotStarted
	}
	if err != nil {
		return 0, fmt.Errorf("failed to lock where the stats are counted to: %w", err)
	}
	return takes.Position(position), nil //nolint:gosec // CHECK (start >= 0), position >= start.
}

func statsOf(ctx context.Context, tx *sql.Tx, accounts []players.AccountID) (map[players.AccountID]players.Stats, error) {
	current := make(map[players.AccountID]players.Stats, len(accounts))
	if len(accounts) == 0 {
		return current, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT account_id, tiles_taken, streak_current, streak_best, streak_last_day, messages_sent
		FROM stats WHERE account_id = ANY($1::uuid[])
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the stats of a batch: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account                       uuid.UUID
			tiles, streak, best, messages int64
			lastDay                       sql.NullTime
		)
		if err := rows.Scan(&account, &tiles, &streak, &best, &lastDay, &messages); err != nil {
			return nil, fmt.Errorf("failed to read the stats of an account: %w", err)
		}
		var day players.Day
		if lastDay.Valid {
			day = players.DayOf(lastDay.Time)
		}
		current[players.AccountID(account)] = players.StatsOf(
			players.AccountID(account),
			uint64(tiles),                         //nolint:gosec // CHECK (tiles_taken >= 0).
			players.StreakOf(uint32(streak), day), //nolint:gosec // CHECK (streak_current >= 0), and one a day.
			uint32(best),                          //nolint:gosec // as above.
			uint64(messages),                      //nolint:gosec // CHECK (messages_sent >= 0).
		)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the stats of a batch: %w", err)
	}
	return current, nil
}

func saveTakes(ctx context.Context, tx *sql.Tx, tallied []players.Stats) error {
	if len(tallied) == 0 {
		return nil
	}

	var (
		ids      = make([]string, len(tallied))
		tiles    = make([]int64, len(tallied))
		streaks  = make([]int64, len(tallied))
		bests    = make([]int64, len(tallied))
		lastDays = make([]string, len(tallied))
	)
	for i, stats := range tallied {
		ids[i] = stats.Account().String()
		tiles[i] = int64(stats.TilesTaken()) //nolint:gosec // one a tile taken: never past int64.
		streaks[i] = int64(stats.Streak().Days())
		bests[i] = int64(stats.StreakBest())
		lastDays[i] = stats.Streak().LastDay().String()
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stats (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
		SELECT account_id, tiles_taken, streak_current, streak_best, NULLIF(streak_last_day, '')::date
		FROM unnest($1::uuid[], $2::bigint[], $3::bigint[], $4::bigint[], $5::text[])
			AS counted (account_id, tiles_taken, streak_current, streak_best, streak_last_day)
		ON CONFLICT (account_id) DO UPDATE SET
			tiles_taken = excluded.tiles_taken,
			streak_current = excluded.streak_current,
			streak_best = excluded.streak_best,
			streak_last_day = excluded.streak_last_day
	`, pq.Array(ids), pq.Array(tiles), pq.Array(streaks), pq.Array(bests), pq.Array(lastDays)); err != nil {
		return fmt.Errorf("failed to save the stats of a batch: %w", err)
	}
	return nil
}

func bigint(position takes.Position) int64 {
	return int64(position) //nolint:gosec // a position comes from a bigint.
}
