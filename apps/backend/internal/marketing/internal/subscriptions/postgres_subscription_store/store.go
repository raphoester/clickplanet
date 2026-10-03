package postgres_subscription_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ subscriptions.Store = (*Store)(nil)

const columns = `account_id, address, state, consent, asked_at, withdrawn_at`

func (s *Store) Subscription(ctx context.Context, account subscriptions.AccountID) (*subscriptions.Subscription, error) {
	found, err := scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM subscriptions WHERE account_id = $1`, uuid.UUID(account)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, subscriptions.ErrNoSubscription
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the subscription: %w", err)
	}
	return found, nil
}

func (s *Store) SubscriptionsTo(ctx context.Context, address subscriptions.Address) ([]*subscriptions.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+columns+` FROM subscriptions WHERE address = $1`, string(address))
	if err != nil {
		return nil, fmt.Errorf("failed to read the subscriptions to an address: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := []*subscriptions.Subscription{}
	for rows.Next() {
		subscription, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read a subscription: %w", err)
		}
		found = append(found, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the subscriptions to an address: %w", err)
	}
	return found, nil
}

func (s *Store) Save(ctx context.Context, subscription *subscriptions.Subscription) error {
	var withdrawnAt sql.NullTime
	if !subscription.WithdrawnAt.IsZero() {
		withdrawnAt = sql.NullTime{Time: subscription.WithdrawnAt.UTC(), Valid: true}
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (`+columns+`) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account_id) DO UPDATE SET
			address = excluded.address, state = excluded.state, consent = excluded.consent,
			asked_at = excluded.asked_at, withdrawn_at = excluded.withdrawn_at
	`, uuid.UUID(subscription.Account), string(subscription.Address), string(subscription.State),
		string(subscription.Consent), subscription.AskedAt.UTC(), withdrawnAt); err != nil {
		return fmt.Errorf("failed to save the subscription: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, account subscriptions.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the subscription: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scan(row scanner) (*subscriptions.Subscription, error) {
	var (
		account                 uuid.UUID
		address, state, consent string
		askedAt                 time.Time
		withdrawnAt             sql.NullTime
	)
	if err := row.Scan(&account, &address, &state, &consent, &askedAt, &withdrawnAt); err != nil {
		return nil, err //nolint:wrapcheck // each caller names what it was reading.
	}

	subscription := &subscriptions.Subscription{
		Account: subscriptions.AccountID(account), Address: subscriptions.Address(address),
		State: subscriptions.State(state), Consent: subscriptions.Consent(consent), AskedAt: askedAt.UTC(),
	}
	if withdrawnAt.Valid {
		subscription.WithdrawnAt = withdrawnAt.Time.UTC()
	}
	return subscription, nil
}
