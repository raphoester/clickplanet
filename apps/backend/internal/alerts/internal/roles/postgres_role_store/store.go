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

func (s *Store) RoleOf(ctx context.Context, key roles.Key) (roles.Kept, error) {
	var (
		id    string
		color int
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT role_id, color FROM roles WHERE key = $1`, string(key)).Scan(&id, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return roles.Kept{}, roles.ErrNoRole
	}
	if err != nil {
		return roles.Kept{}, fmt.Errorf("failed to read a role: %w", err)
	}
	return roles.Kept{ID: discord.RoleID(id), Color: discord.Color(color)}, nil
}

func (s *Store) SaveRole(ctx context.Context, key roles.Key, kept roles.Kept) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO roles (key, role_id, color) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET role_id = EXCLUDED.role_id, color = EXCLUDED.color
	`, string(key), string(kept.ID), int(kept.Color)); err != nil {
		return fmt.Errorf("failed to upsert a role: %w", err)
	}
	return nil
}

func (s *Store) Keys(ctx context.Context) ([]roles.Key, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT key FROM roles ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("failed to list the roles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var keys []roles.Key
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to read a role's key: %w", err)
		}
		keys = append(keys, roles.Key(key))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list the roles: %w", err)
	}
	return keys, nil
}

func (s *Store) ForgetRole(ctx context.Context, key roles.Key) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM roles WHERE key = $1`, string(key)); err != nil {
		return fmt.Errorf("failed to forget a role: %w", err)
	}
	return nil
}
