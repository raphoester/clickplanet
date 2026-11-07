package postgres_raid_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ raids.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, mode raids.RaidMode) error {
	saved := mode.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO raid_mode (started_at, ends_at, reason) VALUES ($1, $2, $3)
		ON CONFLICT (single) DO UPDATE SET
			started_at = EXCLUDED.started_at, ends_at = EXCLUDED.ends_at, reason = EXCLUDED.reason
	`, saved.StartedAt.UTC(), saved.EndsAt.UTC(), string(saved.Reason)); err != nil {
		return fmt.Errorf("failed to keep the raid mode: %w", err)
	}
	return nil
}

func (s *Store) Current(ctx context.Context) (raids.RaidMode, error) {
	var (
		startedAt, endsAt time.Time
		reason            string
	)
	err := s.db.On(ctx).QueryRowContext(ctx, `SELECT started_at, ends_at, reason FROM raid_mode`).
		Scan(&startedAt, &endsAt, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return raids.RaidMode{}, raids.ErrNoRaidMode
	}
	if err != nil {
		return raids.RaidMode{}, fmt.Errorf("failed to read the raid mode: %w", err)
	}
	mode, err := raids.Snapshot{StartedAt: startedAt, EndsAt: endsAt, Reason: moderation.Reason(reason)}.Restore()
	if err != nil {
		return raids.RaidMode{}, fmt.Errorf("failed to restore the raid mode: %w", err)
	}
	return mode, nil
}
