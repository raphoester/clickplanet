package postgres_announcement_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ announcements.Storage = (*Store)(nil)

func (s *Store) Append(ctx context.Context, announcement announcements.Announcement) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO announcements (id, kind, payload, announced_at)
		VALUES ($1, $2::announcement_kind, $3, $4)
	`,
		uuid.UUID(announcement.ID()), string(announcement.Kind()), string(announcement.Payload()), announcement.At().UTC(),
	); err != nil {
		return fmt.Errorf("failed to insert an announcement: %w", err)
	}
	return nil
}

func (s *Store) AppendOnce(ctx context.Context, announcement announcements.Announcement) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO announcements (id, kind, payload, announced_at)
		VALUES ($1, $2::announcement_kind, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`,
		uuid.UUID(announcement.ID()), string(announcement.Kind()), string(announcement.Payload()), announcement.At().UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("failed to insert an announcement: %w", err)
	}
	appended, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to count the announcement inserted: %w", err)
	}
	return appended == 1, nil
}

func (s *Store) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM announcements WHERE announced_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old announcements: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to count deleted announcements: %w", err)
	}
	return deleted, nil
}
