package postgres_checkout_store

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ checkouts.Store = (*Store)(nil)

func (s *Store) Save(ctx context.Context, checkout checkouts.Checkout) error {
	kept := checkout.TakeSnapshot()
	kind, target := kept.Target.Kept()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO checkouts (session_id, target_kind, target, opened_at) VALUES ($1, $2::target_kind, $3, $4)
		ON CONFLICT (session_id) DO NOTHING
	`, kept.Session.String(), string(kind), target, kept.OpenedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the checkout %s: %w", kept.Session, err)
	}
	return nil
}

func (s *Store) OfTarget(ctx context.Context, target orders.Target) ([]checkouts.Checkout, error) {
	kind, id := target.Kept()
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT session_id, opened_at FROM checkouts WHERE target_kind = $1::target_kind AND target = $2
		ORDER BY opened_at, session_id
	`, string(kind), id)
	if err != nil {
		return nil, fmt.Errorf("failed to read the checkouts of %s: %w", id, err)
	}
	var opened []checkouts.Checkout
	for rows.Next() {
		kept := checkouts.Snapshot{Target: target}
		var session string
		if err := rows.Scan(&session, &kept.OpenedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to read a checkout of %s: %w", id, err)
		}
		kept.Session, kept.OpenedAt = checkouts.SessionID(session), kept.OpenedAt.UTC()
		checkout, err := kept.Restore()
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to restore a checkout kept: %w", err)
		}
		opened = append(opened, checkout)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("failed to read the checkouts of %s: %w", id, err)
	}
	return opened, nil
}
