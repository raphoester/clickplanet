// Package postgres_account_store keeps accounts, their identities and their sessions in the auth schema.
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

var _ accounts.Store = (*Store)(nil)

func (s *Store) FindSession(ctx context.Context, tokenHash []byte) (*accounts.Session, error) {
	session := accounts.Session{TokenHash: tokenHash}
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, extended_at, expires_at,
		       EXISTS (SELECT 1 FROM identities WHERE identities.account_id = sessions.account_id)
		FROM sessions WHERE token_hash = $1
	`, tokenHash).Scan(&session.Account, &session.ExtendedAt, &session.ExpiresAt, &session.Linked)
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
		if err := insertAccount(ctx, tx, session.Account, session.ExtendedAt); err != nil {
			return err
		}
		return insertSession(ctx, tx, session)
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

		return markSeen(ctx, tx, session.Account, session.ExtendedAt)
	})
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("failed to delete the session: %w", err)
	}
	return nil
}

func (s *Store) DeleteSessions(ctx context.Context, account uuid.UUID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = $1`, account); err != nil {
		return fmt.Errorf("failed to delete the account's sessions: %w", err)
	}
	return nil
}

func (s *Store) FindAccount(ctx context.Context, account uuid.UUID) (*accounts.Account, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE id = $1)`, account).Scan(&exists); err != nil {
		return nil, fmt.Errorf("failed to select the account: %w", err)
	}
	if !exists {
		return nil, accounts.ErrAccountNotFound
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT provider, subject, account_id, COALESCE(email, ''), email_verified, linked_at
		FROM identities WHERE account_id = $1 ORDER BY linked_at, provider
	`, account)
	if err != nil {
		return nil, fmt.Errorf("failed to select the account's identities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := &accounts.Account{ID: account, Identities: []accounts.Identity{}}
	for rows.Next() {
		identity, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		found.Identities = append(found.Identities, *identity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the account's identities: %w", err)
	}
	return found, nil
}

func (s *Store) FindIdentity(ctx context.Context, provider string, subject string) (*accounts.Identity, error) {
	identity, err := scanIdentity(s.db.QueryRowContext(ctx, `
		SELECT provider, subject, account_id, COALESCE(email, ''), email_verified, linked_at
		FROM identities WHERE provider = $1 AND subject = $2
	`, provider, subject))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrIdentityNotFound
	}
	return identity, err
}

// SaveSignIn writes the account, the identity and the session, and deletes the replaced session, in one transaction.
func (s *Store) SaveSignIn(ctx context.Context, signIn accounts.SignIn) error {
	session := signIn.Session
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if signIn.NewAccount {
			if err := insertAccount(ctx, tx, session.Account, session.ExtendedAt); err != nil {
				return err
			}
		}
		if signIn.Identity != nil {
			if err := insertIdentity(ctx, tx, signIn.Identity); err != nil {
				return err
			}
		}
		if signIn.Replaces != nil {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, signIn.Replaces); err != nil {
				return fmt.Errorf("failed to delete the replaced session: %w", err)
			}
		}
		if err := insertSession(ctx, tx, session); err != nil {
			return err
		}
		return markSeen(ctx, tx, session.Account, session.ExtendedAt)
	})
}

// DeleteAccount deletes the account row; its identities and sessions go with it by cascade.
func (s *Store) DeleteAccount(ctx context.Context, account uuid.UUID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, account); err != nil {
		return fmt.Errorf("failed to delete the account: %w", err)
	}
	return nil
}

func (s *Store) PruneGuests(ctx context.Context, idleSince time.Time, limit int) (int, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM accounts WHERE id IN (
			SELECT id FROM accounts
			WHERE last_seen_at < $1
			  AND NOT EXISTS (SELECT 1 FROM identities WHERE identities.account_id = accounts.id)
			LIMIT $2
		)
	`, idleSince.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the idle guests: %w", err)
	}
	pruned, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the pruned guests: %w", err)
	}
	return int(pruned), nil
}

func insertAccount(ctx context.Context, tx *sql.Tx, account uuid.UUID, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)
	`, account, at.UTC()); err != nil {
		return fmt.Errorf("failed to insert the account: %w", err)
	}
	return nil
}

func insertSession(ctx context.Context, tx *sql.Tx, session *accounts.Session) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, account_id, created_at, extended_at, expires_at) VALUES ($1, $2, $3, $3, $4)
	`, session.TokenHash, session.Account, session.ExtendedAt.UTC(), session.ExpiresAt.UTC()); err != nil {
		return fmt.Errorf("failed to insert the session: %w", err)
	}
	return nil
}

// insertIdentity answers accounts.ErrIdentityTaken when the provider's user is already linked, and the caller's transaction rolls back.
func insertIdentity(ctx context.Context, tx *sql.Tx, identity *accounts.Identity) error {
	var email sql.NullString
	if identity.Email != "" {
		email = sql.NullString{String: identity.Email, Valid: true}
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO identities (provider, subject, account_id, email, email_verified, linked_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (provider, subject) DO NOTHING
	`, identity.Provider, identity.Subject, identity.Account, email, identity.EmailVerified, identity.LinkedAt.UTC())
	if err != nil {
		return fmt.Errorf("failed to insert the identity: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count the inserted identities: %w", err)
	}
	if inserted == 0 {
		return accounts.ErrIdentityTaken
	}
	return nil
}

func markSeen(ctx context.Context, tx *sql.Tx, account uuid.UUID, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET last_seen_at = $2 WHERE id = $1`, account, at.UTC()); err != nil {
		return fmt.Errorf("failed to mark the account seen: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIdentity(row scanner) (*accounts.Identity, error) {
	var identity accounts.Identity
	err := row.Scan(&identity.Provider, &identity.Subject, &identity.Account, &identity.Email, &identity.EmailVerified, &identity.LinkedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan the identity: %w", err)
	}
	identity.LinkedAt = identity.LinkedAt.UTC()
	return &identity, nil
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
