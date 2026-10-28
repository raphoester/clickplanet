package postgres_appearance_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// One kept again is replaced.
func (s *Store) Keep(ctx context.Context, named []appearances.Appearance) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		for _, appearance := range named {
			platforms := make([]string, 0, len(appearance.Post.Platforms))
			for _, platform := range appearance.Post.Platforms {
				platforms = append(platforms, string(platform))
			}
			if _, err := s.db.On(ctx).ExecContext(ctx, `
				INSERT INTO appearances (post_id, account, title, platforms, scheduled_at) VALUES ($1, $2, $3, $4::platform[], $5)
				ON CONFLICT (post_id, account) DO UPDATE
				SET title = EXCLUDED.title, platforms = EXCLUDED.platforms, scheduled_at = EXCLUDED.scheduled_at
			`, string(appearance.Post.ID), uuid.UUID(appearance.Account), appearance.Post.Title, pq.Array(platforms),
				appearance.Post.ScheduledAt); err != nil {
				return fmt.Errorf("failed to keep that %s appears: %w", appearance.Post.ID, err)
			}
		}
		return nil
	})
}

func (s *Store) ScheduledBefore(ctx context.Context, at time.Time) ([]appearances.Key, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT post_id, account FROM appearances WHERE scheduled_at < $1`, at)
	if err != nil {
		return nil, fmt.Errorf("failed to find the appearances before %s: %w", at.Format(time.RFC3339), err)
	}
	defer func() { _ = rows.Close() }()

	var keys []appearances.Key
	for rows.Next() {
		var (
			post    string
			account uuid.UUID
		)
		if err := rows.Scan(&post, &account); err != nil {
			return nil, fmt.Errorf("failed to read an appearance: %w", err)
		}
		keys = append(keys, appearances.Key{Post: appearances.PostID(post), Account: casts.AccountID(account)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to find the appearances before %s: %w", at.Format(time.RFC3339), err)
	}
	return keys, nil
}

func (s *Store) Forget(ctx context.Context, key appearances.Key) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM appearances WHERE post_id = $1 AND account = $2`,
		string(key.Post), uuid.UUID(key.Account)); err != nil {
		return fmt.Errorf("failed to forget an appearance: %w", err)
	}
	return nil
}

func (s *Store) ForgetAccount(ctx context.Context, account casts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM appearances WHERE account = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the account's appearances: %w", err)
	}
	return nil
}
