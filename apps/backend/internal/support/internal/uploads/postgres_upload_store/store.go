package postgres_upload_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ uploads.Storage = (*Store)(nil)

// One lock for every upload: the bytes held are one number.
const reserveLock = 0x75706c64

func (s *Store) Held(ctx context.Context) (int64, error) {
	return held(ctx, s.db.On(ctx))
}

func held(ctx context.Context, db cppg.Querier) (int64, error) {
	var bytes int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size_bytes), 0) FROM uploads`).Scan(&bytes); err != nil {
		return 0, fmt.Errorf("failed to read the bytes held: %w", err)
	}
	return bytes, nil
}

func (s *Store) Reserve(ctx context.Context, upload uploads.Upload, check func(held int64) error) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, reserveLock); err != nil {
			return fmt.Errorf("failed to lock the uploads: %w", err)
		}
		bytes, err := held(ctx, tx)
		if err != nil {
			return err
		}
		if err := check(bytes); err != nil {
			return err
		}
		return write(ctx, tx, upload.TakeSnapshot())
	})
}

func (s *Store) Unsent(ctx context.Context, cutoff time.Time, limit int) ([]uploads.Upload, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT u.id, u.size_bytes, u.granted_at FROM uploads u
		WHERE u.granted_at < $1 AND NOT EXISTS (SELECT 1 FROM attachments a WHERE a.id = u.id)
		ORDER BY u.granted_at, u.id
		LIMIT $2
	`, cutoff.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read the uploads no note names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var unsent []uploads.Upload
	for rows.Next() {
		var (
			id   uuid.UUID
			size int64
			at   time.Time
		)
		if err := rows.Scan(&id, &size, &at); err != nil {
			return nil, fmt.Errorf("failed to read an upload: %w", err)
		}
		upload, err := uploads.Snapshot{ID: notes.AttachmentID(id), Size: size, At: at}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the upload %s: %w", id, err)
		}
		unsent = append(unsent, upload)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the uploads no note names: %w", err)
	}
	return unsent, nil
}

func (s *Store) Save(ctx context.Context, saved ...uploads.Upload) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, upload := range saved {
			if err := write(ctx, tx, upload.TakeSnapshot()); err != nil {
				return err
			}
		}
		return nil
	})
}

func write(ctx context.Context, tx cppg.Querier, kept uploads.Snapshot) error {
	if kept.Discarded {
		if _, err := tx.ExecContext(ctx, `DELETE FROM uploads WHERE id = $1`, uuid.UUID(kept.ID)); err != nil {
			return fmt.Errorf("failed to forget an upload: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO uploads (id, size_bytes, granted_at) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET size_bytes = EXCLUDED.size_bytes, granted_at = EXCLUDED.granted_at
	`, uuid.UUID(kept.ID), kept.Size, kept.At.UTC()); err != nil {
		return fmt.Errorf("failed to upsert an upload: %w", err)
	}
	return nil
}
