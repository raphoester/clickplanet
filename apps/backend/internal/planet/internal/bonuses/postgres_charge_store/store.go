// Package postgres_charge_store keeps the in-memory charges between boots: one row per account.
package postgres_charge_store

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"time"

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

func (s *Store) Load(ctx context.Context, visit func(bonuses.Holder, bonuses.Hand)) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT account::text, bomb_until, enclose_until, spread_until, spread_clicks FROM charges`)
	if err != nil {
		return fmt.Errorf("failed to read the charges: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account               string
			bomb, enclose, spread sql.NullTime
			spreadClicks          int
		)
		if err := rows.Scan(&account, &bomb, &enclose, &spread, &spreadClicks); err != nil {
			return fmt.Errorf("failed to scan a hand: %w", err)
		}
		visit(bonuses.Holder(account), bonuses.Hand{
			Bomb:         timeOf(bomb),
			Enclose:      timeOf(enclose),
			Spread:       timeOf(spread),
			SpreadClicks: spreadClicks,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read the charges: %w", err)
	}

	return nil
}

// Save deletes the hands that hold nothing and writes the others, in one transaction.
func (s *Store) Save(ctx context.Context, hands map[bonuses.Holder]bonuses.Hand) error {
	var (
		gone                     []string
		accounts                 []string
		bombs, encloses, spreads []sql.NullTime
		spreadClicks             []int64
	)
	for _, holder := range slices.Sorted(maps.Keys(hands)) {
		hand := hands[holder]
		if hand == (bonuses.Hand{}) {
			gone = append(gone, string(holder))
			continue
		}
		accounts = append(accounts, string(holder))
		bombs = append(bombs, nullTime(hand.Bomb))
		encloses = append(encloses, nullTime(hand.Enclose))
		spreads = append(spreads, nullTime(hand.Spread))
		spreadClicks = append(spreadClicks, int64(hand.SpreadClicks))
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
			INSERT INTO charges (account, bomb_until, enclose_until, spread_until, spread_clicks)
			SELECT * FROM unnest($1::uuid[], $2::timestamptz[], $3::timestamptz[], $4::timestamptz[], $5::integer[])
			ON CONFLICT (account) DO UPDATE SET
				bomb_until = EXCLUDED.bomb_until,
				enclose_until = EXCLUDED.enclose_until,
				spread_until = EXCLUDED.spread_until,
				spread_clicks = EXCLUDED.spread_clicks
		`, pq.Array(accounts), pq.Array(bombs), pq.Array(encloses), pq.Array(spreads), pq.Array(spreadClicks)); err != nil {
			return fmt.Errorf("failed to write the hands: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the charges: %w", err)
	}

	return nil
}

// nullTime stores no charge of a kind as NULL.
func nullTime(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at, Valid: !at.IsZero()}
}

func timeOf(at sql.NullTime) time.Time {
	if !at.Valid {
		return time.Time{}
	}

	return at.Time.UTC()
}
