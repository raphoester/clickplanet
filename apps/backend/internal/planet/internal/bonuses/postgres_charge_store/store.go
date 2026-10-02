// Package postgres_charge_store keeps the in-memory charges between boots: one row per account.
package postgres_charge_store

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"

	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ inmemory_charge_storage.Persistence = (*Store)(nil)

func (s *Store) Load(ctx context.Context, visit func(bonuses.Holder, bonuses.Held)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT account::text, refill, bomb, enclosures, spread_clicks FROM charges`)
	if err != nil {
		return fmt.Errorf("failed to read the charges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account string
			held    bonuses.Held
		)
		if err := rows.Scan(&account, &held.Refill, &held.Bomb, &held.Enclosures, &held.SpreadClicks); err != nil {
			return fmt.Errorf("failed to scan a hand: %w", err)
		}
		visit(bonuses.Holder(account), held)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the charges: %w", err)
	}

	return nil
}

// Save deletes the hands that hold nothing and writes the others, in one transaction.
func (s *Store) Save(ctx context.Context, hands map[bonuses.Holder]bonuses.Held) error {
	var (
		gone                     []string
		accounts                 []string
		refills, bombs           []bool
		enclosures, spreadClicks []int64
	)
	for _, holder := range slices.Sorted(maps.Keys(hands)) {
		held := hands[holder]
		if held.Empty() {
			gone = append(gone, string(holder))
			continue
		}
		accounts = append(accounts, string(holder))
		refills = append(refills, held.Refill)
		bombs = append(bombs, held.Bomb)
		enclosures = append(enclosures, int64(held.Enclosures))
		spreadClicks = append(spreadClicks, int64(held.SpreadClicks))
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to save the charges: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if len(gone) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM charges WHERE account = ANY($1::uuid[])`, pq.Array(gone)); err != nil {
			return fmt.Errorf("failed to delete spent hands: %w", err)
		}
	}

	if len(accounts) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO charges (account, refill, bomb, enclosures, spread_clicks)
			SELECT * FROM unnest($1::uuid[], $2::boolean[], $3::boolean[], $4::integer[], $5::integer[])
			ON CONFLICT (account) DO UPDATE SET
				refill = EXCLUDED.refill,
				bomb = EXCLUDED.bomb,
				enclosures = EXCLUDED.enclosures,
				spread_clicks = EXCLUDED.spread_clicks
		`, pq.Array(accounts), pq.Array(refills), pq.Array(bombs), pq.Array(enclosures), pq.Array(spreadClicks)); err != nil {
			return fmt.Errorf("failed to write the hands: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the charges: %w", err)
	}

	return nil
}
