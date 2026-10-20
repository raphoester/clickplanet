package postgres_admin_session_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ adminsignin.Store = (*Store)(nil)

func (s *Store) Candidate(ctx context.Context, provider string, subject string) (adminsignin.Candidate, error) {
	var (
		account  uuid.UUID
		operates bool
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT accounts.id, accounts.operator
		FROM identities JOIN accounts ON accounts.id = identities.account_id
		WHERE identities.provider = $1::identity_provider AND identities.subject = $2
	`, provider, subject).Scan(&account, &operates)
	if errors.Is(err, sql.ErrNoRows) {
		return adminsignin.Candidate{}, adminsignin.ErrUnknownIdentity
	}
	if err != nil {
		return adminsignin.Candidate{}, fmt.Errorf("failed to read who the identity is: %w", err)
	}

	return adminsignin.CandidateOf(accounts.AccountID(account), operates), nil
}

func (s *Store) SaveSession(ctx context.Context, session adminsignin.Snapshot) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_sessions (token_hash, account_id, created_at, expires_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (token_hash) DO UPDATE SET account_id = $2, created_at = $3, expires_at = $4
	`, []byte(session.TokenHash), uuid.UUID(session.Account), session.CreatedAt.UTC(), session.ExpiresAt.UTC())
	if err != nil {
		return fmt.Errorf("failed to save the admin session: %w", err)
	}
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash accounts.TokenHash) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash = $1`, []byte(tokenHash)); err != nil {
		return fmt.Errorf("failed to delete the admin session: %w", err)
	}
	return nil
}
