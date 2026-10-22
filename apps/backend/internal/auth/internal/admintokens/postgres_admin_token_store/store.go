package postgres_admin_token_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ admintokens.Store = (*Store)(nil)

func (s *Store) SaveTokens(ctx context.Context, tokens []admintokens.Snapshot) error {
	for _, token := range tokens {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO admin_tokens (token_hash, kind, account_id, client_id, client, created_at, expires_at)
			VALUES ($1, $2::admin_token_kind, $3, $4, $5, $6, $7)
		`, []byte(token.Hash), string(token.Kind), uuid.UUID(token.Holder.Account), token.Holder.ClientID, token.Holder.Client,
			token.CreatedAt.UTC(), token.ExpiresAt.UTC())
		if err != nil {
			return fmt.Errorf("failed to save an admin token: %w", err)
		}
	}
	return nil
}

func (s *Store) SpendRefresh(ctx context.Context, hash accounts.TokenHash, now time.Time) (admintokens.Holder, error) {
	var (
		account          uuid.UUID
		clientID, client string
	)
	err := s.db.QueryRowContext(ctx, `
		DELETE FROM admin_tokens
		WHERE token_hash = $1 AND kind = 'refresh' AND expires_at > $2
		RETURNING account_id, client_id, client
	`, []byte(hash), now.UTC()).Scan(&account, &clientID, &client)
	if errors.Is(err, sql.ErrNoRows) {
		return admintokens.Holder{}, admintokens.ErrUnknownToken
	}
	if err != nil {
		return admintokens.Holder{}, fmt.Errorf("failed to spend the refresh token: %w", err)
	}

	return admintokens.Holder{Account: accounts.AccountID(account), ClientID: clientID, Client: client}, nil
}

func (s *Store) DeleteLapsed(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM admin_tokens WHERE expires_at <= $1`, now.UTC()); err != nil {
		return fmt.Errorf("failed to delete the lapsed admin tokens: %w", err)
	}
	return nil
}
