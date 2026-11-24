package postgres_renewal_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// "rnwl": a subscription Stripe tells of before any row of it is kept is locked by its id.
const renewalLock = 0x726e776c

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ renewals.Store = (*Store)(nil)

const columns = `id, account_id, plan::text, target_kind::text, target, agreed_at, state::text, period_ends_at,
	decided_at`

func (s *Store) Renewal(ctx context.Context, id renewals.SubscriptionID) (renewals.Renewal, error) {
	tx := s.db.On(ctx)
	// Stripe tells a subscription's events in no order and at once: the second waits for the first's transaction.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1, hashtext($2))`, renewalLock, id.String()); err != nil {
		return renewals.Renewal{}, fmt.Errorf("failed to lock the renewal: %w", err)
	}
	renewal, err := renewalOf(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM renewals WHERE id = $1`, id.String()).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return renewals.Renewal{}, fmt.Errorf("%s: %w", id, renewals.ErrUnknownRenewal)
	}
	return renewal, err
}

func (s *Store) Bought(ctx context.Context, buyer orders.AccountID) ([]renewals.Renewal, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT `+columns+` FROM renewals WHERE account_id = $1
		ORDER BY period_ends_at, id`, uuid.UUID(buyer))
	if err != nil {
		return nil, fmt.Errorf("failed to read the renewals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var bought []renewals.Renewal
	for rows.Next() {
		renewal, err := renewalOf(rows.Scan)
		if err != nil {
			return nil, err
		}
		bought = append(bought, renewal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the renewals: %w", err)
	}
	return bought, nil
}

func renewalOf(scan func(dest ...any) error) (renewals.Renewal, error) {
	var kept renewals.Snapshot
	var id, plan, kind, target, state string
	var account uuid.UUID
	if err := scan(&id, &account, &plan, &kind, &target, &kept.Order.AgreedAt, &state, &kept.PeriodEndsAt,
		&kept.DecidedAt); err != nil {
		return renewals.Renewal{}, fmt.Errorf("failed to read a renewal: %w", err)
	}
	var err error
	kept.Order.Target, err = orders.TargetOf(orders.TargetKind(kind), target)
	if err != nil {
		return renewals.Renewal{}, fmt.Errorf("the renewal %s: %w", id, err)
	}
	kept.ID, kept.Order.Buyer, kept.Order.Plan = renewals.SubscriptionID(id), orders.AccountID(account), plans.Plan(plan)
	kept.State, kept.Order.AgreedAt = renewals.State(state), kept.Order.AgreedAt.UTC()
	kept.PeriodEndsAt, kept.DecidedAt = kept.PeriodEndsAt.UTC(), kept.DecidedAt.UTC()
	renewal, err := kept.Restore()
	if err != nil {
		return renewals.Renewal{}, fmt.Errorf("failed to restore a renewal kept: %w", err)
	}
	return renewal, nil
}

func (s *Store) Save(ctx context.Context, renewal renewals.Renewal) error {
	kept := renewal.TakeSnapshot()
	kind, target := kept.Order.Target.Kept()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO renewals (id, account_id, plan, target_kind, target, agreed_at, state, period_ends_at, decided_at)
		VALUES ($1, $2, $3::plan, $4::target_kind, $5, $6, $7::renewal_state, $8, $9)
		ON CONFLICT (id) DO UPDATE SET state = EXCLUDED.state, period_ends_at = EXCLUDED.period_ends_at,
			decided_at = EXCLUDED.decided_at
	`, kept.ID.String(), uuid.UUID(kept.Order.Buyer), string(kept.Order.Plan), string(kind), target,
		kept.Order.AgreedAt.UTC(), string(kept.State), kept.PeriodEndsAt.UTC(), kept.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the renewal %s: %w", kept.ID, err)
	}
	return nil
}
