// Package postgres_ledger_store keeps the in-memory ledger between boots: one row per take, plus its marks.
package postgres_ledger_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ inmemory_ledger_storage.Persistence = (*Store)(nil)

// Load visits every take in position order, then reads the marks.
func (s *Store) Load(ctx context.Context, visit func(inmemory_ledger_storage.Stored)) (inmemory_ledger_storage.Marks, error) {
	marks := inmemory_ledger_storage.Marks{Forgotten: map[string]ledger.Position{}}

	if err := s.loadTakes(ctx, visit); err != nil {
		return marks, err
	}

	// No row is a ledger never flushed: head 0.
	var head int64
	err := s.db.QueryRowContext(ctx, `SELECT head FROM ledger_head`).Scan(&head)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return marks, fmt.Errorf("failed to read the ledger head: %w", err)
	}
	marks.Head = ledger.Position(head) //nolint:gosec // CHECK (head >= 0).

	rows, err := s.db.QueryContext(ctx, `SELECT scope, before_position FROM ledger_forgotten`)
	if err != nil {
		return marks, fmt.Errorf("failed to read forgotten scopes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			scope  string
			before int64
		)
		if err := rows.Scan(&scope, &before); err != nil {
			return marks, fmt.Errorf("failed to scan a forgotten scope: %w", err)
		}
		marks.Forgotten[scope] = ledger.Position(before) //nolint:gosec // CHECK (before_position >= 0).
	}

	if err := rows.Err(); err != nil {
		return marks, fmt.Errorf("failed to read forgotten scopes: %w", err)
	}

	return marks, nil
}

func (s *Store) loadTakes(ctx context.Context, visit func(inmemory_ledger_storage.Stored)) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT position, tile, scope, country, previous, taken_at FROM ledger_takes ORDER BY position`)
	if err != nil {
		return fmt.Errorf("failed to read takes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			position int64
			take     ledger.Taking
			at       time.Time
		)
		if err := rows.Scan(&position, &take.Tile, &take.Scope, &take.Country, &take.Previous, &at); err != nil {
			return fmt.Errorf("failed to scan a take: %w", err)
		}
		take.At = at.UTC()
		visit(inmemory_ledger_storage.Stored{Position: ledger.Position(position), Taking: take}) //nolint:gosec // CHECK (position >= 0).
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read takes: %w", err)
	}

	return nil
}

// Save copies the takes in, deletes those before the head, and moves the marks, in one transaction.
func (s *Store) Save(ctx context.Context, changes inmemory_ledger_storage.Changes) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save the ledger: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	head := int64(changes.Marks.Head) //nolint:gosec // a position fits a bigint.

	// Rows at or past From are this flush's takes from a commit whose answer was lost.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM ledger_takes WHERE position >= $1 OR position < $2`,
		int64(changes.From), head, //nolint:gosec // a position fits a bigint.
	); err != nil {
		return fmt.Errorf("failed to delete takes: %w", err)
	}

	if err := copyTakes(ctx, tx, changes); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_head (head) VALUES ($1)
		ON CONFLICT (singleton) DO UPDATE SET head = EXCLUDED.head
	`, head); err != nil {
		return fmt.Errorf("failed to write the ledger head: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM ledger_forgotten WHERE before_position <= $1`, head); err != nil {
		return fmt.Errorf("failed to delete forgotten scopes: %w", err)
	}

	if len(changes.Marks.Forgotten) > 0 {
		scopes := slices.Sorted(maps.Keys(changes.Marks.Forgotten))
		befores := make([]int64, len(scopes))
		for i, scope := range scopes {
			befores[i] = int64(changes.Marks.Forgotten[scope]) //nolint:gosec // a position fits a bigint.
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ledger_forgotten (scope, before_position)
			SELECT * FROM unnest($1::text[], $2::bigint[])
			ON CONFLICT (scope) DO UPDATE SET before_position = GREATEST(ledger_forgotten.before_position, EXCLUDED.before_position)
		`, pq.Array(scopes), pq.Array(befores)); err != nil {
			return fmt.Errorf("failed to write forgotten scopes: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the ledger: %w", err)
	}

	return nil
}

// copyTakes streams the takes through COPY, so an import of millions of takes is one statement.
func copyTakes(ctx context.Context, tx *sql.Tx, changes inmemory_ledger_storage.Changes) error {
	stmt, err := tx.PrepareContext(ctx,
		pq.CopyIn("ledger_takes", "position", "tile", "scope", "country", "previous", "taken_at"))
	if err != nil {
		return fmt.Errorf("failed to start copying takes: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for take := range changes.Takes {
		if _, err := stmt.ExecContext(ctx,
			int64(take.Position), //nolint:gosec // a position fits a bigint.
			int64(take.Taking.Tile),
			take.Taking.Scope,
			take.Taking.Country,
			take.Taking.Previous,
			take.Taking.At,
		); err != nil {
			return fmt.Errorf("failed to copy a take: %w", err)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to copy takes: %w", err)
	}

	return nil
}
