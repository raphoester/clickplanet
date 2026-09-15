// Package postgres_account_store keeps accounts and their sessions in the auth schema.
package postgres_account_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

func (s *Store) FindSession(ctx context.Context, tokenHash []byte) (accounts.Session, bool, error) {
	var session accounts.Session
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, extended_at, expires_at FROM sessions WHERE token_hash = $1
	`, tokenHash).Scan(&session.Account, &session.ExtendedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return accounts.Session{}, false, nil
	}
	if err != nil {
		return accounts.Session{}, false, fmt.Errorf("failed to find a session: %w", err)
	}

	session.ExtendedAt = session.ExtendedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	return session, true, nil
}

// ExtendSession moves the session's expiry and marks its account seen, in one transaction.
func (s *Store) ExtendSession(ctx context.Context, tokenHash []byte, expiresAt, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var account uuid.UUID
		err := tx.QueryRowContext(ctx, `
			UPDATE sessions SET extended_at = $2, expires_at = $3 WHERE token_hash = $1 RETURNING account_id
		`, tokenHash, now.UTC(), expiresAt.UTC()).Scan(&account)
		if err != nil {
			return fmt.Errorf("failed to extend a session: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET last_seen_at = $2 WHERE id = $1`, account, now.UTC()); err != nil {
			return fmt.Errorf("failed to mark an account seen: %w", err)
		}
		return nil
	})
}

// CreateGuest writes a new account and its first session, in one transaction.
func (s *Store) CreateGuest(ctx context.Context, account uuid.UUID, tokenHash []byte, expiresAt, now time.Time) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)
		`, account, now.UTC()); err != nil {
			return fmt.Errorf("failed to insert an account: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (token_hash, account_id, created_at, extended_at, expires_at) VALUES ($1, $2, $3, $3, $4)
		`, tokenHash, account, now.UTC(), expiresAt.UTC()); err != nil {
			return fmt.Errorf("failed to insert a session: %w", err)
		}
		return nil
	})
}

func (s *Store) inTx(ctx context.Context, do func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err //nolint:wrapcheck // cppg already names what failed.
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
