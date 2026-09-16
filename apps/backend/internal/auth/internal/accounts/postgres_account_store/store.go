// Package postgres_account_store keeps accounts and their sessions in the auth schema.
package postgres_account_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ accounts.Sessions = (*Store)(nil)

func (s *Store) FindSession(ctx context.Context, tokenHash []byte) (*accounts.Session, error) {
	session := accounts.Session{TokenHash: tokenHash}
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, extended_at, expires_at FROM sessions WHERE token_hash = $1
	`, tokenHash).Scan(&session.Account, &session.ExtendedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select the session: %w", err)
	}

	session.ExtendedAt = session.ExtendedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	return &session, nil
}

// CreateGuest writes the account and its first session in one transaction.
func (s *Store) CreateGuest(ctx context.Context, session *accounts.Session) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)
		`, session.Account, session.ExtendedAt.UTC()); err != nil {
			return fmt.Errorf("failed to insert the account: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (token_hash, account_id, created_at, extended_at, expires_at) VALUES ($1, $2, $3, $3, $4)
		`, session.TokenHash, session.Account, session.ExtendedAt.UTC(), session.ExpiresAt.UTC()); err != nil {
			return fmt.Errorf("failed to insert the session: %w", err)
		}
		return nil
	})
}

// SaveSession writes the session's expiry and marks its account seen at its last extension, in one transaction.
func (s *Store) SaveSession(ctx context.Context, session *accounts.Session) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE sessions SET extended_at = $2, expires_at = $3 WHERE token_hash = $1
		`, session.TokenHash, session.ExtendedAt.UTC(), session.ExpiresAt.UTC())
		if err != nil {
			return fmt.Errorf("failed to update the session: %w", err)
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to count the updated sessions: %w", err)
		}
		if updated == 0 {
			return accounts.ErrSessionNotFound
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE accounts SET last_seen_at = $2 WHERE id = $1
		`, session.Account, session.ExtendedAt.UTC()); err != nil {
			return fmt.Errorf("failed to mark the account seen: %w", err)
		}
		return nil
	})
}

func (s *Store) inTx(ctx context.Context, do func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin a transaction: %w", err)
	}

	if err := do(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}
