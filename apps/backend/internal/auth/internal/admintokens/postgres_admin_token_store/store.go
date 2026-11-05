package postgres_admin_token_store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

const columns = 8

func (s *Store) SaveTokens(ctx context.Context, tokens ...admintokens.Snapshot) error {
	if len(tokens) == 0 {
		return nil
	}

	rows := make([]string, 0, len(tokens))
	args := make([]any, 0, columns*len(tokens))
	for i, token := range tokens {
		n := i * columns
		rows = append(rows, fmt.Sprintf("($%d, $%d::admin_token_kind, $%d, $%d, $%d, $%d, $%d, $%d)",
			n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8))
		args = append(args, []byte(token.Hash), string(token.Kind), uuid.UUID(token.Holder.Account),
			string(token.Holder.ClientID), token.Holder.Client, string(token.Connection), token.CreatedAt.UTC(),
			token.ExpiresAt.UTC())
	}

	_, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO admin_tokens (token_hash, kind, account_id, client_id, client, connection_id, created_at, expires_at)
		VALUES `+strings.Join(rows, ", ")+`
		ON CONFLICT (token_hash) DO UPDATE SET kind = EXCLUDED.kind, account_id = EXCLUDED.account_id,
			client_id = EXCLUDED.client_id, client = EXCLUDED.client, connection_id = EXCLUDED.connection_id,
			created_at = EXCLUDED.created_at, expires_at = EXCLUDED.expires_at
	`, args...)
	if err != nil {
		return fmt.Errorf("failed to save the admin tokens: %w", err)
	}
	return nil
}

func (s *Store) Connection(ctx context.Context, hash accounts.TokenHash) ([]*admintokens.Token, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT token_hash, kind::text, account_id, client_id, client, connection_id, created_at, expires_at
		FROM admin_tokens
		WHERE connection_id = (SELECT connection_id FROM admin_tokens WHERE token_hash = $1)
		FOR UPDATE
	`, []byte(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to read the connection: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var connection []*admintokens.Token
	for rows.Next() {
		var (
			kept                         admintokens.Snapshot
			keptHash                     []byte
			kind, clientID, connectionID string
			account                      uuid.UUID
			createdAt, expiresAt         time.Time
		)
		if err := rows.Scan(&keptHash, &kind, &account, &clientID, &kept.Holder.Client, &connectionID, &createdAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("failed to read an admin token: %w", err)
		}
		kept.Hash, kept.Kind, kept.Connection = keptHash, admintokens.Kind(kind), admintokens.ConnectionID(connectionID)
		kept.Holder.Account, kept.Holder.ClientID = accounts.AccountID(account), admintokens.ClientID(clientID)
		kept.CreatedAt, kept.ExpiresAt = createdAt.UTC(), expiresAt.UTC()

		token, err := kept.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore an admin token: %w", err)
		}
		connection = append(connection, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the connection: %w", err)
	}
	return connection, nil
}
