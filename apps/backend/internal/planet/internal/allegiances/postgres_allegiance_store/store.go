package postgres_allegiance_store

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/inmemory_allegiance_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ inmemory_allegiance_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(allegiances.Account, allegiances.Snapshot)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT account::text, flag, changed_at FROM allegiances`)
	if err != nil {
		return fmt.Errorf("failed to read the allegiances: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account   string
			flag      string
			changedAt sql.NullTime
		)
		if err := rows.Scan(&account, &flag, &changedAt); err != nil {
			return fmt.Errorf("failed to scan an allegiance: %w", err)
		}
		visit(allegiances.Account(account), allegiances.Snapshot{Flag: flag, ChangedAt: changedAt.Time})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the allegiances: %w", err)
	}

	return nil
}

func (s *Store) Save(ctx context.Context, kept map[allegiances.Account]allegiances.Snapshot) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save the allegiances: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, account := range slices.Sorted(maps.Keys(kept)) {
		allegiance := kept[account]
		changedAt := sql.NullTime{Time: allegiance.ChangedAt.UTC(), Valid: !allegiance.ChangedAt.IsZero()}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO allegiances (account, flag, changed_at) VALUES ($1, $2, $3)
			ON CONFLICT (account) DO UPDATE SET flag = EXCLUDED.flag, changed_at = EXCLUDED.changed_at
		`, string(account), allegiance.Flag, changedAt); err != nil {
			return fmt.Errorf("failed to write the allegiance of %s: %w", account, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the allegiances: %w", err)
	}

	return nil
}
