// Package postgres_ban_store keeps the chat's bans in the chat schema: one row per silenced member.
package postgres_ban_store

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

// All is every ban, in no order: the set it fills is a map.
func (s *Store) All(ctx context.Context) ([]bans.Ban, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag, banned_at, reason FROM bans`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat bans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var all []bans.Ban
	for rows.Next() {
		var ban bans.Ban
		if err := rows.Scan(&ban.AuthorTag, &ban.BannedAt, &ban.Reason); err != nil {
			return nil, fmt.Errorf("failed to scan a chat ban: %w", err)
		}
		ban.BannedAt = ban.BannedAt.UTC()
		all = append(all, ban)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the chat bans: %w", err)
	}

	return all, nil
}

// Upsert rewrites the reason of a ban already running, and keeps whatever time it carries.
func (s *Store) Upsert(ctx context.Context, ban bans.Ban) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO bans (tag, banned_at, reason)
		VALUES ($1, $2, $3)
		ON CONFLICT (tag) DO UPDATE SET banned_at = EXCLUDED.banned_at, reason = EXCLUDED.reason
	`, ban.AuthorTag, ban.BannedAt.UTC(), ban.Reason); err != nil {
		return fmt.Errorf("failed to insert a chat ban: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, tag string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM bans WHERE tag = $1`, tag); err != nil {
		return fmt.Errorf("failed to delete a chat ban: %w", err)
	}
	return nil
}
