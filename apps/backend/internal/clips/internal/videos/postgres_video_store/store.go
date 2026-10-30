package postgres_video_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/videos"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// A video kept again takes its new name and caption.
func (s *Store) Keep(ctx context.Context, video videos.Video) error {
	kept := video.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO videos (video_id, name, caption, media_type, size_bytes, kept_at) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (video_id) DO UPDATE SET name = excluded.name, caption = excluded.caption
	`, uuid.UUID(kept.File.ID), kept.Name, kept.Caption, string(kept.File.MediaType), kept.File.Size, kept.KeptAt); err != nil {
		return fmt.Errorf("failed to keep the video %s: %w", kept.File.ID, err)
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, id videos.VideoID) error {
	res, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM videos WHERE video_id = $1`, uuid.UUID(id))
	if err != nil {
		return fmt.Errorf("failed to remove the video %s: %w", id, err)
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the video %s was removed: %w", id, err)
	}
	if removed == 0 {
		return fmt.Errorf("%w: %s", videos.ErrNoVideo, id)
	}
	return nil
}
