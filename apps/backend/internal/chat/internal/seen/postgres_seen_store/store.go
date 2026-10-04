package postgres_seen_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ seen.Storage = (*Store)(nil)

func (s *Store) SeenUntil(ctx context.Context, account messages.AccountID) (time.Time, error) {
	var until time.Time
	err := s.db.QueryRowContext(ctx, `SELECT seen_until FROM seen WHERE account_id = $1`, uuid.UUID(account)).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read what the account has seen: %w", err)
	}
	return until.UTC(), nil
}

func (s *Store) SaveSeen(ctx context.Context, account messages.AccountID, until time.Time) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO seen (account_id, seen_until)
		VALUES ($1, $2)
		ON CONFLICT (account_id) DO UPDATE SET seen_until = GREATEST(seen.seen_until, EXCLUDED.seen_until)
	`, uuid.UUID(account), until.UTC()); err != nil {
		return fmt.Errorf("failed to save what the account has seen: %w", err)
	}
	return nil
}

func (s *Store) DeleteSeen(ctx context.Context, account messages.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM seen WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete what the account has seen: %w", err)
	}
	return nil
}
