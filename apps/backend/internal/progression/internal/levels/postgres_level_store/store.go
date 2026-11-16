package postgres_level_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ levels.Store = (*Store)(nil)

func (s *Store) Levels(ctx context.Context, asked []accounts.AccountID) (levels.ByAccount, error) {
	kept := levels.ByAccount{}
	if len(asked) == 0 {
		return kept, nil
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT account_id, level FROM levels WHERE account_id = ANY($1::uuid[])`,
		pq.Array(idsOf(asked)))
	if err != nil {
		return nil, fmt.Errorf("failed to read the levels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account uuid.UUID
			level   int64
		)
		if err := rows.Scan(&account, &level); err != nil {
			return nil, fmt.Errorf("failed to read a level: %w", err)
		}
		kept[accounts.AccountID(account)] = levels.Level(level) //nolint:gosec // an integer column with a CHECK (>= 1).
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the levels: %w", err)
	}
	return kept, nil
}

func (s *Store) Raise(ctx context.Context, account accounts.AccountID, level levels.Level, at time.Time) (levels.Level, error) {
	var previous levels.Level
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for {
			inserted, err := tx.ExecContext(ctx, `
				INSERT INTO levels (account_id, level, reached_at) VALUES ($1, $2, $3) ON CONFLICT (account_id) DO NOTHING
			`, uuid.UUID(account), int64(level), at.UTC())
			if err != nil {
				return fmt.Errorf("failed to keep the first level: %w", err)
			}
			count, err := inserted.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to count the first level: %w", err)
			}
			if count == 1 {
				return nil
			}

			var kept int64
			err = tx.QueryRowContext(ctx, `SELECT level FROM levels WHERE account_id = $1 FOR UPDATE`, uuid.UUID(account)).Scan(&kept)
			if errors.Is(err, sql.ErrNoRows) {
				continue // forgotten since the insert met it: insert again.
			}
			if err != nil {
				return fmt.Errorf("failed to lock the level kept: %w", err)
			}
			previous = levels.Level(kept) //nolint:gosec // an integer column with a CHECK (>= 1).
			if previous >= level {
				return nil
			}
			if _, err := tx.ExecContext(ctx, `UPDATE levels SET level = $2, reached_at = $3 WHERE account_id = $1`,
				uuid.UUID(account), int64(level), at.UTC()); err != nil {
				return fmt.Errorf("failed to raise the level: %w", err)
			}
			return nil
		}
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // each step named it.
	}
	return previous, nil
}

func (s *Store) RaiseAll(ctx context.Context, raised levels.ByAccount, at time.Time) error {
	return s.write(ctx, raised, at, `WHERE kept.level < excluded.level`)
}

func (s *Store) Set(ctx context.Context, set levels.ByAccount, at time.Time) error {
	return s.write(ctx, set, at, ``)
}

func (s *Store) write(ctx context.Context, written levels.ByAccount, at time.Time, onlyIf string) error {
	if len(written) == 0 {
		return nil
	}
	ids := make([]string, 0, len(written))
	values := make([]int64, 0, len(written))
	for account, level := range written {
		ids = append(ids, account.String())
		values = append(values, int64(level))
	}

	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO levels AS kept (account_id, level, reached_at)
		SELECT written.account_id, written.level, $3
		FROM unnest($1::uuid[], $2::integer[]) AS written (account_id, level)
		ON CONFLICT (account_id) DO UPDATE SET level = excluded.level, reached_at = excluded.reached_at
		`+onlyIf, pq.Array(ids), pq.Array(values), at.UTC()); err != nil {
		return fmt.Errorf("failed to write the levels: %w", err)
	}
	return nil
}

func (s *Store) Forget(ctx context.Context, forgotten ...accounts.AccountID) error {
	if len(forgotten) == 0 {
		return nil
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM levels WHERE account_id = ANY($1::uuid[])`,
		pq.Array(idsOf(forgotten))); err != nil {
		return fmt.Errorf("failed to forget the levels: %w", err)
	}
	return nil
}

func idsOf(asked []accounts.AccountID) []string {
	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}
	return ids
}
