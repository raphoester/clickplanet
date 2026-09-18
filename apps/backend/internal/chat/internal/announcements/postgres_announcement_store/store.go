// Package postgres_announcement_store keeps every chat announcement in chat.announcements. It is the chat's only
// copy: every read and write goes to postgres.
package postgres_announcement_store

import (
	"context"
	"fmt"
	"slices"
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
		VALUES ($1, $2, $3, $4)
	`,
		uuid.UUID(announcement.ID), string(announcement.Kind), string(announcement.Payload), announcement.At.UTC(),
	); err != nil {
		return fmt.Errorf("failed to insert an announcement: %w", err)
	}
	return nil
}

// Recent is the newest limit announcements made at or after since, oldest first.
func (s *Store) Recent(ctx context.Context, since time.Time, limit int) ([]announcements.Announcement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload, announced_at
		FROM announcements
		WHERE announced_at >= $1
		ORDER BY announced_at DESC, id DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read announcements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var recent []announcements.Announcement
	for rows.Next() {
		var (
			announcement announcements.Announcement
			id           uuid.UUID
			kind         string
			payload      []byte
		)
		if err := rows.Scan(&id, &kind, &payload, &announcement.At); err != nil {
			return nil, fmt.Errorf("failed to scan an announcement: %w", err)
		}
		announcement.ID = announcements.AnnouncementID(id)
		announcement.Kind = announcements.Kind(kind)
		announcement.Payload = payload
		announcement.At = announcement.At.UTC()
		recent = append(recent, announcement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read announcements: %w", err)
	}

	slices.Reverse(recent)
	return recent, nil
}

// DeleteBefore removes every announcement made before cutoff and says how many it removed.
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
