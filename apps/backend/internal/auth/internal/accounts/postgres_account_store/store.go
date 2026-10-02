package postgres_account_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

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

func (s *Store) Session(ctx context.Context, tokenHash accounts.TokenHash) (*accounts.Session, error) {
	session := accounts.Session{TokenHash: tokenHash}
	var account uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id, extended_at, expires_at,
		       EXISTS (SELECT 1 FROM identities WHERE identities.account_id = sessions.account_id)
		FROM sessions WHERE token_hash = $1
	`, tokenHash).Scan(&account, &session.ExtendedAt, &session.ExpiresAt, &session.Linked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select the session: %w", err)
	}

	session.Account = accounts.AccountID(account)
	session.ExtendedAt = session.ExtendedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	return &session, nil
}

func (s *Store) CreateGuest(ctx context.Context, session *accounts.Session) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if err := insertAccount(ctx, tx, session.Account, session.ExtendedAt); err != nil {
			return err
		}
		return insertSession(ctx, tx, session)
	})
}

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

func (s *Store) DeleteSession(ctx context.Context, tokenHash accounts.TokenHash) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("failed to delete the session: %w", err)
	}
	return nil
}

func (s *Store) DeleteSessions(ctx context.Context, account accounts.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's sessions: %w", err)
	}
	return nil
}

func (s *Store) Account(ctx context.Context, account accounts.AccountID) (*accounts.Account, error) {
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, `SELECT created_at FROM accounts WHERE id = $1`, uuid.UUID(account)).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select the account: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT provider, subject, account_id, COALESCE(email, ''), email_verified, linked_at
		FROM identities WHERE account_id = $1 ORDER BY linked_at, provider
	`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to select the account's identities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	found := &accounts.Account{ID: account, CreatedAt: createdAt.UTC(), Identities: []accounts.Identity{}}
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

func (s *Store) CreationDates(ctx context.Context, asked []accounts.AccountID) (map[accounts.AccountID]time.Time, error) {
	dates := make(map[accounts.AccountID]time.Time, len(asked))
	if len(asked) == 0 {
		return dates, nil
	}

	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, created_at FROM accounts WHERE id = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to select the creation dates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account   uuid.UUID
			createdAt time.Time
		)
		if err := rows.Scan(&account, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to read a creation date: %w", err)
		}
		dates[accounts.AccountID(account)] = createdAt.UTC()
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the creation dates: %w", err)
	}
	return dates, nil
}

func (s *Store) Identity(ctx context.Context, provider string, subject string) (*accounts.Identity, error) {
	identity, err := scanIdentity(s.db.QueryRowContext(ctx, `
		SELECT provider, subject, account_id, COALESCE(email, ''), email_verified, linked_at
		FROM identities WHERE provider = $1 AND subject = $2
	`, provider, subject))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrIdentityNotFound
	}
	return identity, err
}

func (s *Store) AccountOfEmail(ctx context.Context, address string) (*accounts.Account, error) {
	var account uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		SELECT account_id FROM identities
		WHERE email_verified AND lower(email) = lower($1)
		ORDER BY linked_at, provider LIMIT 1
	`, address).Scan(&account)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, accounts.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select the account of the address: %w", err)
	}
	return s.Account(ctx, accounts.AccountID(account))
}

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

func (s *Store) DeleteAccount(ctx context.Context, account accounts.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account: %w", err)
	}
	return nil
}

func (s *Store) PruneGuests(ctx context.Context, idleSince time.Time, limit int) ([]accounts.AccountID, error) {
	rows, err := s.db.QueryContext(ctx, `
		DELETE FROM accounts WHERE id IN (
			SELECT id FROM accounts
			WHERE last_seen_at < $1
			  AND NOT EXISTS (SELECT 1 FROM identities WHERE identities.account_id = accounts.id)
			LIMIT $2
		)
		RETURNING id
	`, idleSince.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to prune the idle guests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var pruned []accounts.AccountID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to read a pruned guest: %w", err)
		}
		pruned = append(pruned, accounts.AccountID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the pruned guests: %w", err)
	}
	return pruned, nil
}

func insertAccount(ctx context.Context, tx *sql.Tx, account accounts.AccountID, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO accounts (id, created_at, last_seen_at) VALUES ($1, $2, $2)
	`, uuid.UUID(account), at.UTC()); err != nil {
		return fmt.Errorf("failed to insert the account: %w", err)
	}
	return nil
}

func insertSession(ctx context.Context, tx *sql.Tx, session *accounts.Session) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, account_id, created_at, extended_at, expires_at) VALUES ($1, $2, $3, $3, $4)
	`, session.TokenHash, uuid.UUID(session.Account), session.ExtendedAt.UTC(), session.ExpiresAt.UTC()); err != nil {
		return fmt.Errorf("failed to insert the session: %w", err)
	}
	return nil
}

func insertIdentity(ctx context.Context, tx *sql.Tx, identity *accounts.Identity) error {
	var email sql.NullString
	if identity.Email != "" {
		email = sql.NullString{String: identity.Email, Valid: true}
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO identities (provider, subject, account_id, email, email_verified, linked_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (provider, subject) DO NOTHING
	`, identity.Provider, identity.Subject, uuid.UUID(identity.Account), email, identity.EmailVerified, identity.LinkedAt.UTC())
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

func markSeen(ctx context.Context, tx *sql.Tx, account accounts.AccountID, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET last_seen_at = $2 WHERE id = $1`, uuid.UUID(account), at.UTC()); err != nil {
		return fmt.Errorf("failed to mark the account seen: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIdentity(row scanner) (*accounts.Identity, error) {
	var identity accounts.Identity
	var account uuid.UUID
	err := row.Scan(&identity.Provider, &identity.Subject, &account, &identity.Email, &identity.EmailVerified, &identity.LinkedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan the identity: %w", err)
	}
	identity.Account = accounts.AccountID(account)
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
