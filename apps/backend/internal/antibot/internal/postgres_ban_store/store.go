package postgres_ban_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var _ bans.Store = (*Store)(nil)

const (
	scopesLock   = 0x62616e73
	accountsLock = 0x61636e74
)

func NewScopes(db cppg.Conn) *Store {
	return &Store{db: db, lock: scopesLock, queries: queries{
		record: `
			SELECT b.flags, b.offences, b.expires_at, b.last_flagged_at,
				p.pardoned_at, p.flags, p.offence, p.banned_until, p.expires_at
			FROM bans b
			LEFT JOIN LATERAL (
				SELECT pardoned_at, flags, offence, banned_until, expires_at FROM pardons
				WHERE pardons.scope = b.scope ORDER BY pardoned_at DESC LIMIT 1
			) p ON true
			WHERE b.scope = $1
		`,
		running: `SELECT count(*) FROM bans WHERE expires_at > $1`,
		save: `
			INSERT INTO bans (scope, flags, offences, expires_at, last_flagged_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (scope) DO UPDATE SET
				flags = excluded.flags,
				offences = excluded.offences,
				expires_at = excluded.expires_at,
				last_flagged_at = excluded.last_flagged_at
		`,
		savePardon: `
			INSERT INTO pardons (scope, pardoned_at, flags, offence, banned_until, expires_at) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (scope, pardoned_at) DO UPDATE SET
				flags = excluded.flags,
				offence = excluded.offence,
				banned_until = excluded.banned_until,
				expires_at = excluded.expires_at
		`,
	}}
}

func NewAccounts(db cppg.Conn) *Store {
	return &Store{db: db, lock: accountsLock, queries: queries{
		record: `
			SELECT b.flags, b.offences, b.expires_at, b.last_flagged_at,
				p.pardoned_at, p.flags, p.offence, p.banned_until, p.expires_at
			FROM account_bans b
			LEFT JOIN LATERAL (
				SELECT pardoned_at, flags, offence, banned_until, expires_at FROM account_pardons
				WHERE account_pardons.account = b.account ORDER BY pardoned_at DESC LIMIT 1
			) p ON true
			WHERE b.account = $1
		`,
		running: `SELECT count(*) FROM account_bans WHERE expires_at > $1`,
		save: `
			INSERT INTO account_bans (account, flags, offences, expires_at, last_flagged_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account) DO UPDATE SET
				flags = excluded.flags,
				offences = excluded.offences,
				expires_at = excluded.expires_at,
				last_flagged_at = excluded.last_flagged_at
		`,
		savePardon: `
			INSERT INTO account_pardons (account, pardoned_at, flags, offence, banned_until, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (account, pardoned_at) DO UPDATE SET
				flags = excluded.flags,
				offence = excluded.offence,
				banned_until = excluded.banned_until,
				expires_at = excluded.expires_at
		`,
	}}
}

type Store struct {
	db      cppg.Conn
	lock    int64
	queries queries
}

type queries struct {
	record     string
	running    string
	save       string
	savePardon string
}

func (s *Store) Record(ctx context.Context, key string) (bans.Record, bool, error) {
	return s.recordOf(ctx, s.db.On(ctx), key)
}

func (s *Store) recordOf(ctx context.Context, db cppg.Querier, key string) (bans.Record, bool, error) {
	record := bans.Record{Key: key}

	var (
		flagged                              sql.NullTime
		pardonedAt, liftedUntil, pardonUntil sql.NullTime
		liftedFlags, liftedOffence           sql.NullInt64
	)
	err := db.QueryRowContext(ctx, s.queries.record, key).Scan(
		&record.Flags, &record.Offences, &record.ExpiresAt, &flagged,
		&pardonedAt, &liftedFlags, &liftedOffence, &liftedUntil, &pardonUntil,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return record, false, nil
	}
	if err != nil {
		return bans.Record{}, false, fmt.Errorf("failed to read the ban: %w", err)
	}

	record.ExpiresAt = record.ExpiresAt.UTC()
	if flagged.Valid {
		record.LastFlaggedAt = flagged.Time.UTC()
	}
	if pardonedAt.Valid {
		record.Pardon = bans.Pardon{
			At: pardonedAt.Time.UTC(),
			Lifted: bans.Sentence{
				Flags:   int(liftedFlags.Int64),
				Offence: int(liftedOffence.Int64),
				Until:   liftedUntil.Time.UTC(),
			},
			Until: pardonUntil.Time.UTC(),
		}
	}

	return record, true, nil
}

func (s *Store) Running(ctx context.Context, now time.Time) (int, error) {
	var running int
	if err := s.db.On(ctx).QueryRowContext(ctx, s.queries.running, now).Scan(&running); err != nil {
		return 0, fmt.Errorf("failed to count the running bans: %w", err)
	}
	return running, nil
}

func (s *Store) Change(
	ctx context.Context,
	key string,
	change func(record bans.Record, found bool) (bans.Record, bool),
) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, s.lock, key); err != nil {
			return fmt.Errorf("failed to lock the ban: %w", err)
		}

		record, found, err := s.recordOf(ctx, tx, key)
		if err != nil {
			return err
		}

		next, keep := change(record, found)
		if !keep {
			return nil
		}

		if _, err := tx.ExecContext(ctx, s.queries.save, key, next.Flags, next.Offences, next.ExpiresAt,
			sql.NullTime{Time: next.LastFlaggedAt, Valid: !next.LastFlaggedAt.IsZero()}); err != nil {
			return fmt.Errorf("failed to save the ban: %w", err)
		}

		if !next.Pardon.At.IsZero() {
			pardon := next.Pardon
			if _, err := tx.ExecContext(ctx, s.queries.savePardon, key, pardon.At,
				pardon.Lifted.Flags, pardon.Lifted.Offence, pardon.Lifted.Until, pardon.Until); err != nil {
				return fmt.Errorf("failed to save the pardon: %w", err)
			}
		}
		return nil
	})
}
