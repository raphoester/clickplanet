package postgres_worn_title_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ wearing.Store = (*Store)(nil)

func (s *Store) Choice(ctx context.Context, account players.AccountID) (titles.ID, error) {
	var title string
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT title FROM worn_titles WHERE account_id = $1`, uuid.UUID(account)).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the title chosen: %w", err)
	}
	return titles.ID(title), nil
}

func (s *Store) Wear(ctx context.Context, account players.AccountID, title titles.ID, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO worn_titles (account_id, title, worn_at) VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET title = excluded.title, worn_at = excluded.worn_at
	`, uuid.UUID(account), string(title), at.UTC()); err != nil {
		return fmt.Errorf("failed to wear the title: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM worn_titles WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's worn title: %w", err)
	}
	return nil
}
