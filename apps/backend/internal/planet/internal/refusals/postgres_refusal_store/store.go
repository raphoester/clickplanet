package postgres_refusal_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const refusalLock = 0x72656675

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ refusals.Store = (*Store)(nil)

func (s *Store) Refusals(ctx context.Context, callers []refusals.Caller) (map[refusals.Caller]refusals.Refusal, error) {
	kept := make(map[refusals.Caller]refusals.Refusal, len(callers))
	if len(callers) == 0 {
		return kept, nil
	}

	kinds, keys := make([]string, 0, len(callers)), make([]string, 0, len(callers))
	for _, caller := range callers {
		kind, key := keyOf(caller)
		kinds, keys = append(kinds, kind), append(keys, key)
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT r.kind::text, r.caller, r.refused_until, r.decided_at
		FROM refused_callers r
		JOIN unnest($1::text[], $2::text[]) AS c (kind, caller) ON r.kind = c.kind::caller_kind AND r.caller = c.caller
	`, kinds, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to read the refusals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var kind, key string
		var row refusals.Snapshot
		if err := rows.Scan(&kind, &key, &row.Until, &row.DecidedAt); err != nil {
			return nil, fmt.Errorf("failed to scan a refusal: %w", err)
		}
		caller, err := callerOf(kind, key)
		if err != nil {
			return nil, err
		}
		kept[caller] = refusals.Snapshot{Until: row.Until.UTC(), DecidedAt: row.DecidedAt.UTC()}.Restore()
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the refusals: %w", err)
	}
	return kept, nil
}

func (s *Store) Change(ctx context.Context, caller refusals.Caller, change func(refusals.Refusal) refusals.Refusal) error {
	kind, key := keyOf(caller)

	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, refusalLock, kind+":"+key); err != nil {
			return fmt.Errorf("failed to lock the refusal: %w", err)
		}

		before, err := keptOf(ctx, tx, caller)
		if err != nil {
			return err
		}

		after := change(before.Restore()).TakeSnapshot()
		if after == before {
			return nil
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO refused_callers (kind, caller, refused_until, decided_at) VALUES ($1::caller_kind, $2, $3, $4)
			ON CONFLICT (kind, caller) DO UPDATE SET refused_until = EXCLUDED.refused_until, decided_at = EXCLUDED.decided_at
		`, kind, key, after.Until.UTC(), after.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to write the refusal: %w", err)
		}
		return nil
	})
}

func keptOf(ctx context.Context, db cppg.Querier, caller refusals.Caller) (refusals.Snapshot, error) {
	kind, key := keyOf(caller)

	var kept refusals.Snapshot
	err := db.QueryRowContext(ctx,
		`SELECT refused_until, decided_at FROM refused_callers WHERE kind = $1::caller_kind AND caller = $2`, kind, key).
		Scan(&kept.Until, &kept.DecidedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return refusals.Snapshot{}, nil
	case err != nil:
		return refusals.Snapshot{}, fmt.Errorf("failed to read the refusal: %w", err)
	}

	return refusals.Snapshot{Until: kept.Until.UTC(), DecidedAt: kept.DecidedAt.UTC()}, nil
}

func keyOf(caller refusals.Caller) (string, string) {
	if caller.Account != "" {
		return "account", caller.Account
	}
	return "scope", caller.Scope
}

func callerOf(kind, key string) (refusals.Caller, error) {
	switch kind {
	case "account":
		return refusals.Caller{Account: key}, nil
	case "scope":
		return refusals.Caller{Scope: key}, nil
	}
	return refusals.Caller{}, fmt.Errorf("a refusal of an unknown kind %q", kind)
}
