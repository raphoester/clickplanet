package postgres_ledger_store

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ inmemory_ledger_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(inmemory_ledger_storage.Stored)) (inmemory_ledger_storage.Marks, error) {
	marks := inmemory_ledger_storage.Marks{Forgotten: map[ledger.Caller]ledger.Position{}}

	head, err := storedHead(ctx, s.db.On(ctx))
	if err != nil {
		return marks, err
	}
	marks.Head = ledger.Position(head) //nolint:gosec // CHECK (head >= 0).

	if err := s.loadEvents(ctx, head, visit); err != nil {
		return marks, err
	}

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
	rows, err := s.db.On(ctx).QueryContext(ctx, query)
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

func storedHead(ctx context.Context, db cppg.Querier) (int64, error) {
	var head int64
	err := db.QueryRowContext(ctx, `SELECT head FROM ledger_head`).Scan(&head)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to read the ledger head: %w", err)
	}
	return head, nil
}

func (s *Store) Relayed(ctx context.Context) (ledger.Position, error) {
	var position int64
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT position FROM ledger_relayed`).Scan(&position)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to read how far the ledger was told: %w", err)
	}
	return ledger.Position(position), nil //nolint:gosec // CHECK (position >= 0).
}

func (s *Store) SaveRelayed(ctx context.Context, position ledger.Position) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO ledger_relayed (position) VALUES ($1)
		ON CONFLICT (singleton) DO UPDATE SET position = excluded.position
	`, int64(position)); err != nil { //nolint:gosec // a ledger position fits in int64.
		return fmt.Errorf("failed to save how far the ledger was told: %w", err)
	}
	return nil
}

func (s *Store) loadEvents(ctx context.Context, head int64, visit func(inmemory_ledger_storage.Stored)) error {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT position, kind, tile, scope, account::text, country, previous, taken_at, payload
		FROM ledger_events
		WHERE position >= $1
		ORDER BY position
	`, head)
	if err != nil {
		return fmt.Errorf("failed to read the ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			position int64
			entry    ledger.Entry
			account  sql.NullString
		)
		if err := rows.Scan(&position, &entry.Kind, &entry.Tile, &entry.Scope, &account,
			&entry.Country, &entry.Previous, &entry.At, &entry.Payload); err != nil {
			return fmt.Errorf("failed to scan an event: %w", err)
		}
		entry.Account = account.String
		entry.At = entry.At.UTC()
		visit(inmemory_ledger_storage.Stored{Position: ledger.Position(position), Entry: entry}) //nolint:gosec // CHECK (position >= 0).
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the ledger: %w", err)
	}

	return nil
}

func (s *Store) Save(ctx context.Context, changes inmemory_ledger_storage.Changes) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		head := int64(changes.Marks.Head) //nolint:gosec // a position fits a bigint.

		kept, err := storedHead(ctx, tx)
		if err != nil {
			return err
		}

		// Rows at or past From come from a flush whose commit answer was lost.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM ledger_events WHERE position >= $1`,
			int64(changes.From), //nolint:gosec // a position fits a bigint.
		); err != nil {
			return fmt.Errorf("failed to delete events: %w", err)
		}

		if err := copyEvents(ctx, tx, changes); err != nil {
			return err
		}

		if head > kept {
			if _, err := tx.ExecContext(ctx,
				`UPDATE ledger_events SET scope = NULL WHERE position >= $1 AND position < $2`, kept, head,
			); err != nil {
				return fmt.Errorf("failed to blank the scopes behind the head: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ledger_head (head) VALUES ($1)
			ON CONFLICT (singleton) DO UPDATE SET head = EXCLUDED.head
		`, head); err != nil {
			return fmt.Errorf("failed to write the ledger head: %w", err)
		}

		return saveForgotten(ctx, tx, changes.Marks, head)
	})
}

func (s *Store) AnonymizeTakes(ctx context.Context, account ledger.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`UPDATE ledger_events SET account = NULL WHERE account = $1`, uuid.UUID(account),
	); err != nil {
		return fmt.Errorf("failed to anonymize an account's takes: %w", err)
	}
	return nil
}

func copyEvents(ctx context.Context, tx cppg.Querier, changes inmemory_ledger_storage.Changes) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("ledger_events",
		"position", "kind", "tile", "scope", "account", "country", "previous", "taken_at", "payload"))
	if err != nil {
		return fmt.Errorf("failed to start copying events: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for stored := range changes.Entries {
		entry := stored.Entry
		if _, err := stmt.ExecContext(ctx,
			int64(stored.Position), //nolint:gosec // a position fits a bigint.
			entry.Kind,
			int64(entry.Tile),
			entry.Scope,
			nullIfEmpty(entry.Account),
			entry.Country,
			entry.Previous,
			entry.At,
			jsonOrNull(entry.Payload),
		); err != nil {
			return fmt.Errorf("failed to copy an event: %w", err)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to copy events: %w", err)
	}

	return nil
}

func saveForgotten(ctx context.Context, tx cppg.Querier, marks inmemory_ledger_storage.Marks, head int64) error {
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

func jsonOrNull(payload []byte) any {
	if payload == nil {
		return nil
	}
	return string(payload)
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
