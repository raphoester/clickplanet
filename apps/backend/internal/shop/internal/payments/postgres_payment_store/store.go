package postgres_payment_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ payments.Store = (*Store)(nil)

const columns = `id, account_id, plan::text, target_kind::text, target, agreed_at, intent, amount, currency,
	state::text, paid_at, decided_at`

func (s *Store) Payment(ctx context.Context, id payments.PaymentID) (payments.Payment, error) {
	return s.one(ctx, `SELECT `+columns+` FROM payments WHERE id = $1 FOR UPDATE`, id.String())
}

func (s *Store) OfIntent(ctx context.Context, intent payments.IntentID) (payments.Payment, error) {
	return s.one(ctx, `SELECT `+columns+` FROM payments WHERE intent = $1 FOR UPDATE`, intent.String())
}

func (s *Store) OfTarget(ctx context.Context, target orders.Target) ([]payments.Payment, error) {
	kind, id := target.Kept()
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT `+columns+` FROM payments
		WHERE target_kind = $1::target_kind AND target = $2 ORDER BY paid_at, id`, string(kind), id)
	if err != nil {
		return nil, fmt.Errorf("failed to read the payments for %s: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	var paid []payments.Payment
	for rows.Next() {
		payment, err := paymentOf(rows.Scan)
		if err != nil {
			return nil, err
		}
		paid = append(paid, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the payments for %s: %w", id, err)
	}
	return paid, nil
}

func (s *Store) one(ctx context.Context, query string, arg string) (payments.Payment, error) {
	payment, err := paymentOf(s.db.On(ctx).QueryRowContext(ctx, query, arg).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return payments.Payment{}, fmt.Errorf("%s: %w", arg, payments.ErrUnknownPayment)
	}
	return payment, err
}

func paymentOf(scan func(dest ...any) error) (payments.Payment, error) {
	var kept payments.Snapshot
	var id, plan, kind, target, intent, state string
	var account uuid.UUID
	err := scan(&id, &account, &plan, &kind, &target,
		&kept.Order.AgreedAt, &intent, &kept.Amount, &kept.Currency, &state, &kept.PaidAt, &kept.DecidedAt)
	if err != nil {
		return payments.Payment{}, fmt.Errorf("failed to read a payment: %w", err)
	}
	kept.Order.Target, err = orders.TargetOf(orders.TargetKind(kind), target)
	if err != nil {
		return payments.Payment{}, fmt.Errorf("the payment %s: %w", id, err)
	}
	kept.ID, kept.Order.Buyer, kept.Order.Plan = payments.PaymentID(id), orders.AccountID(account), plans.Plan(plan)
	kept.Intent, kept.State = payments.IntentID(intent), payments.State(state)
	kept.Order.AgreedAt, kept.PaidAt, kept.DecidedAt = kept.Order.AgreedAt.UTC(), kept.PaidAt.UTC(), kept.DecidedAt.UTC()
	payment, err := kept.Restore()
	if err != nil {
		return payments.Payment{}, fmt.Errorf("failed to restore a payment kept: %w", err)
	}
	return payment, nil
}

func (s *Store) Save(ctx context.Context, payment payments.Payment) error {
	kept := payment.TakeSnapshot()
	kind, target := kept.Order.Target.Kept()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO payments (id, account_id, plan, target_kind, target, agreed_at, intent, amount, currency, state,
			paid_at, decided_at)
		VALUES ($1, $2, $3::plan, $4::target_kind, $5, $6, $7, $8, $9, $10::payment_state, $11, $12)
		ON CONFLICT (id) DO UPDATE SET state = EXCLUDED.state, decided_at = EXCLUDED.decided_at
	`, kept.ID.String(), uuid.UUID(kept.Order.Buyer), string(kept.Order.Plan), string(kind), target,
		kept.Order.AgreedAt.UTC(), kept.Intent.String(), kept.Amount, kept.Currency, string(kept.State),
		kept.PaidAt.UTC(), kept.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the payment %s: %w", kept.ID, err)
	}
	return nil
}
