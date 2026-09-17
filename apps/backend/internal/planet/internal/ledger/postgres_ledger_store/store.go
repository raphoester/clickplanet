// Package postgres_ledger_store keeps the in-memory ledger between boots: one row per take, plus its marks.
package postgres_ledger_store

import (
	"cmp"
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
	marks := inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{}}

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

	if err := s.loadForgotten(ctx, `SELECT scope, before_position FROM ledger_forgotten`, marks.Forgotten,
		func(scope string) ledger.Caller { return ledger.Caller{Scope: scope} }); err != nil {
		return marks, err
	}

	if err := s.loadForgotten(ctx, `SELECT account::text, before_position FROM ledger_forgotten_accounts`, marks.Forgotten,
		func(account string) ledger.Caller { return ledger.Caller{Account: account} }); err != nil {
		return marks, err
	}

	return marks, nil
}

func (s *Store) loadForgotten(
	ctx context.Context,
	query string,
	forgotten map[ledger.Caller]ledger.Position,
	callerOf func(string) ledger.Caller,
) error {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to read forgotten callers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			key    string
			before int64
		)
		if err := rows.Scan(&key, &before); err != nil {
			return fmt.Errorf("failed to scan a forgotten caller: %w", err)
		}
		forgotten[callerOf(key)] = ledger.Position(before) //nolint:gosec // CHECK (before_position >= 0).
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read forgotten callers: %w", err)
	}

	return nil
}

func (s *Store) loadTakes(ctx context.Context, visit func(inmemory_ledger_storage.Stored)) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT position, tile, scope, account::text, country, previous, taken_at FROM ledger_takes ORDER BY position`)
	if err != nil {
		return fmt.Errorf("failed to read takes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			position int64
			take     ledger.Taking
			account  sql.NullString
			at       time.Time
		)
		if err := rows.Scan(&position, &take.Tile, &take.Scope, &account, &take.Country, &take.Previous, &at); err != nil {
			return fmt.Errorf("failed to scan a take: %w", err)
		}
		take.Account = account.String
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

	if err := saveForgotten(ctx, tx, changes.Marks, head); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the ledger: %w", err)
	}

	return nil
}

// copyTakes streams the takes through COPY, so an import of millions of takes is one statement.
func copyTakes(ctx context.Context, tx *sql.Tx, changes inmemory_ledger_storage.Changes) error {
	stmt, err := tx.PrepareContext(ctx,
		pq.CopyIn("ledger_takes", "position", "tile", "scope", "account", "country", "previous", "taken_at"))
	if err != nil {
		return fmt.Errorf("failed to start copying takes: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for take := range changes.Takes {
		if _, err := stmt.ExecContext(ctx,
			int64(take.Position), //nolint:gosec // a position fits a bigint.
			int64(take.Taking.Tile),
			take.Taking.Scope,
			nullIfEmpty(take.Taking.Account),
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

// saveForgotten drops the marks the head passed, and writes the ones set since the last flush.
func saveForgotten(ctx context.Context, tx *sql.Tx, marks inmemory_ledger_storage.Marks, head int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM ledger_forgotten WHERE before_position <= $1`, head); err != nil {
		return fmt.Errorf("failed to delete forgotten scopes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ledger_forgotten_accounts WHERE before_position <= $1`, head); err != nil {
		return fmt.Errorf("failed to delete forgotten accounts: %w", err)
	}

	var (
		scopes, accounts             []string
		scopeBefores, accountBefores []int64
	)
	for _, caller := range slices.SortedFunc(maps.Keys(marks.Forgotten), compareCallers) {
		before := int64(marks.Forgotten[caller]) //nolint:gosec // a position fits a bigint.
		if caller.Account != "" {
			accounts, accountBefores = append(accounts, caller.Account), append(accountBefores, before)
			continue
		}
		scopes, scopeBefores = append(scopes, caller.Scope), append(scopeBefores, before)
	}

	if len(scopes) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ledger_forgotten (scope, before_position)
			SELECT * FROM unnest($1::text[], $2::bigint[])
			ON CONFLICT (scope) DO UPDATE SET before_position = GREATEST(ledger_forgotten.before_position, EXCLUDED.before_position)
		`, pq.Array(scopes), pq.Array(scopeBefores)); err != nil {
			return fmt.Errorf("failed to write forgotten scopes: %w", err)
		}
	}

	if len(accounts) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ledger_forgotten_accounts (account, before_position)
			SELECT * FROM unnest($1::uuid[], $2::bigint[])
			ON CONFLICT (account) DO UPDATE SET
				before_position = GREATEST(ledger_forgotten_accounts.before_position, EXCLUDED.before_position)
		`, pq.Array(accounts), pq.Array(accountBefores)); err != nil {
			return fmt.Errorf("failed to write forgotten accounts: %w", err)
		}
	}

	return nil
}

func compareCallers(a, b ledger.Caller) int {
	return cmp.Or(cmp.Compare(a.Scope, b.Scope), cmp.Compare(a.Account, b.Account))
}

// nullIfEmpty stores a take with no account as NULL, which a uuid column needs.
func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
