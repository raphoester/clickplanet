package postgres_role_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ roles.Store = (*Store)(nil)

func (s *Store) RoleOf(ctx context.Context, key roles.Key) (discord.RoleID, error) {
	var id string
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT role_id FROM roles WHERE key = $1`, string(key)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", roles.ErrNoRole
	}
	if err != nil {
		return "", fmt.Errorf("failed to read a role: %w", err)
	}
	return discord.RoleID(id), nil
}

func (s *Store) SaveRole(ctx context.Context, key roles.Key, id discord.RoleID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO roles (key, role_id) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET role_id = EXCLUDED.role_id
	`, string(key), string(id)); err != nil {
		return fmt.Errorf("failed to upsert a role: %w", err)
	}
	return nil
}
