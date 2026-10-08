package postgres_ban_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ shadowban.Store = (*Store)(nil)

const (
	scopesLock   = 0x62616e73
	accountsLock = 0x61636e74
)

func NewScopes(db cppg.QuerierBeginner) *Store {
	return &Store{db: db, lock: scopesLock, queries: queries{
		record:  `SELECT flags, offences, banned_until, next_flag_at FROM bans WHERE scope = $1`,
		running: `SELECT count(*) FROM bans WHERE banned_until > $1`,
		save: `
			INSERT INTO bans (scope, flags, offences, banned_until, next_flag_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (scope) DO UPDATE SET
				flags = excluded.flags,
				offences = excluded.offences,
				banned_until = excluded.banned_until,
				next_flag_at = excluded.next_flag_at
		`,
	}}
}

func NewAccounts(db cppg.QuerierBeginner) *Store {
	return &Store{db: db, lock: accountsLock, queries: queries{
		record:  `SELECT flags, offences, banned_until, next_flag_at FROM account_bans WHERE account = $1`,
		running: `SELECT count(*) FROM account_bans WHERE banned_until > $1`,
		save: `
			INSERT INTO account_bans (account, flags, offences, banned_until, next_flag_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account) DO UPDATE SET
				flags = excluded.flags,
				offences = excluded.offences,
				banned_until = excluded.banned_until,
				next_flag_at = excluded.next_flag_at
		`,
	}}
}

type Store struct {
	db      cppg.QuerierBeginner
	lock    int64
	queries queries
}

type queries struct {
	record  string
	running string
	save    string
}

func (s *Store) Record(ctx context.Context, key string) (shadowban.Record, bool, error) {
	return s.recordOf(ctx, s.db, key)
}

func (s *Store) recordOf(ctx context.Context, db cppg.Querier, key string) (shadowban.Record, bool, error) {
	record := shadowban.Record{Key: key}

	var next sql.NullTime
	err := db.QueryRowContext(ctx, s.queries.record, key).Scan(&record.Flags, &record.Offences, &record.Until, &next)
	if errors.Is(err, sql.ErrNoRows) {
		return record, false, nil
	}
	if err != nil {
		return shadowban.Record{}, false, fmt.Errorf("failed to read the ban: %w", err)
	}

	record.Until = record.Until.UTC()
	if next.Valid {
		record.NextFlagAt = next.Time.UTC()
	}

	return record, true, nil
}

func (s *Store) Running(ctx context.Context, now time.Time) (int, error) {
	var running int
	if err := s.db.QueryRowContext(ctx, s.queries.running, now).Scan(&running); err != nil {
		return 0, fmt.Errorf("failed to count the running bans: %w", err)
	}
	return running, nil
}

func (s *Store) Change(
	ctx context.Context,
	key string,
	change func(record shadowban.Record, found bool) (shadowban.Record, bool),
) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin changing the ban: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, s.lock, key); err != nil {
		return fmt.Errorf("failed to lock the ban: %w", err)
	}

	record, found, err := s.recordOf(ctx, tx, key)
	if err != nil {
		return err
	}

	next, keep := change(record, found)
	if !keep {
		return tx.Commit() //nolint:wrapcheck // nothing was written, so there is nothing to name.
	}

	if _, err := tx.ExecContext(ctx, s.queries.save, key, next.Flags, next.Offences, next.Until,
		sql.NullTime{Time: next.NextFlagAt, Valid: !next.NextFlagAt.IsZero()}); err != nil {
		return fmt.Errorf("failed to save the ban: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the ban: %w", err)
	}
	return nil
}
