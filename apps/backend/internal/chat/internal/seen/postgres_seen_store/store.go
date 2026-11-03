package postgres_seen_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ seen.Storage = (*Store)(nil)

func (s *Store) SaveSeen(ctx context.Context, account messages.AccountID, room messages.Room, until time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO seen (account_id, room, seen_until)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id, room) DO UPDATE SET seen_until = GREATEST(seen.seen_until, EXCLUDED.seen_until)
	`, uuid.UUID(account), string(room), until.UTC()); err != nil {
		return fmt.Errorf("failed to save what the account has seen: %w", err)
	}
	return nil
}

func (s *Store) DeleteSeen(ctx context.Context, account messages.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM seen WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete what the account has seen: %w", err)
	}
	return nil
}
