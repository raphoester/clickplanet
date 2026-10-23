package postgres_charge_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const chargeLock = 0x63686172

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ bonuses.Store = (*Store)(nil)

func (s *Store) Change(
	ctx context.Context,
	holder bonuses.Holder,
	change func(bonuses.Held) (bonuses.Held, error),
) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin changing the charges: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, chargeLock, string(holder)); err != nil {
		return fmt.Errorf("failed to lock the charges: %w", err)
	}

	before, err := heldBy(ctx, tx, holder)
	if err != nil {
		return err
	}

	after, err := change(before)
	if err != nil {
		return err
	}

	if after != before {
		if err := keep(ctx, tx, holder, after); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the charges: %w", err)
	}
	return nil
}

func (s *Store) Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error) {
	return heldBy(ctx, s.db, holder)
}

func keep(ctx context.Context, tx *sql.Tx, holder bonuses.Holder, held bonuses.Held) error {
	if held.Empty() {
		if _, err := tx.ExecContext(ctx, `DELETE FROM charges WHERE account = $1`, string(holder)); err != nil {
			return fmt.Errorf("failed to delete a spent hand: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO charges (account, refill, bomb, enclosures, spread_clicks, shields) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account) DO UPDATE SET
			refill = EXCLUDED.refill,
			bomb = EXCLUDED.bomb,
			enclosures = EXCLUDED.enclosures,
			spread_clicks = EXCLUDED.spread_clicks,
			shields = EXCLUDED.shields
	`, string(holder), held.Refill, held.Bomb, held.Enclosures, held.SpreadClicks, held.Shields); err != nil {
		return fmt.Errorf("failed to write the hand: %w", err)
	}
	return nil
}

func heldBy(ctx context.Context, db cppg.Querier, holder bonuses.Holder) (bonuses.Held, error) {
	var held bonuses.Held
	err := db.QueryRowContext(ctx,
		`SELECT refill, bomb, enclosures, spread_clicks, shields FROM charges WHERE account = $1`, string(holder)).
		Scan(&held.Refill, &held.Bomb, &held.Enclosures, &held.SpreadClicks, &held.Shields)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return bonuses.Held{}, nil
	case err != nil:
		return bonuses.Held{}, fmt.Errorf("failed to read the hand: %w", err)
	}

	return held, nil
}
