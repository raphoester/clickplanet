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
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO announcements (id, kind, payload, announced_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`,
		uuid.UUID(announcement.ID()), string(announcement.Kind()), string(announcement.Payload()), announcement.At().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert an announcement: %w", err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether an announcement was inserted: %w", err)
	}
	if inserted == 0 {
		return announcements.ErrKept
	}
	return nil
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
