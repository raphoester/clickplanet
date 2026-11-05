package postgres_grant_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// The token is sealed at rest: a copy of the table, or of a backup, joins nobody to the server.
func New(db cppg.Conn, sealer grants.Sealer) *Store {
	return &Store{db: db, sealer: sealer}
}

type Store struct {
	db     cppg.Conn
	sealer grants.Sealer
}

var _ grants.Store = (*Store)(nil)

func (s *Store) SaveGrant(ctx context.Context, grant grants.Grant) error {
	kept := grant.TakeSnapshot()
	sealed, err := s.sealer.Sealed(kept.ID, kept.Token)
	if err != nil {
		return fmt.Errorf("failed to seal the token: %w", err)
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO discord_grants (grant_id, account_id, discord_user_id, sealed_token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.UUID(kept.ID), uuid.UUID(kept.Account), kept.User, sealed, kept.ExpiresAt.UTC()); err != nil {
		return fmt.Errorf("failed to insert the grant: %w", err)
	}
	return nil
}

func (s *Store) Grant(ctx context.Context, id grants.GrantID) (grants.Grant, error) {
	var (
		account   uuid.UUID
		user      string
		sealed    []byte
		expiresAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT account_id, discord_user_id, sealed_token, expires_at FROM discord_grants WHERE grant_id = $1
	`, uuid.UUID(id)).Scan(&account, &user, &sealed, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return grants.Grant{}, grants.ErrGrantNotFound
	}
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to read the grant: %w", err)
	}
	token, err := s.sealer.Opened(id, sealed)
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to open the grant's token: %w", err)
	}
	grant, err := grants.Snapshot{
		ID: id, Account: grants.AccountID(account), User: user, Token: token, ExpiresAt: expiresAt.UTC(),
	}.Restore()
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to restore the grant %s: %w", id, err)
	}
	return grant, nil
}

func (s *Store) PruneGrants(ctx context.Context, expiredBy time.Time, limit int) (int, error) {
	res, err := s.db.On(ctx).ExecContext(ctx, `
		DELETE FROM discord_grants WHERE grant_id IN (
			SELECT grant_id FROM discord_grants WHERE expires_at <= $1 LIMIT $2
		)
	`, expiredBy.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("failed to prune the expired grants: %w", err)
	}
	pruned, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count the pruned grants: %w", err)
	}
	return int(pruned), nil
}
