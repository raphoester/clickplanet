package postgres_grant_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ books.Grants = (*Store)(nil)

func (s *Store) Keep(ctx context.Context, grant books.Grant) (bool, error) {
	keeping, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO grants (grant_id, country, points, granted_at) VALUES ($1, $2, $3, $4) ON CONFLICT (grant_id) DO NOTHING
	`, string(grant.ID), string(grant.Country), int64(grant.Points), grant.At)
	if err != nil {
		return false, fmt.Errorf("failed to keep the grant: %w", err)
	}
	affected, err := keeping.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to keep the grant: %w", err)
	}
	return affected > 0, nil
}

func (s *Store) Since(ctx context.Context, at time.Time) ([]books.Grant, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT grant_id, country, points, granted_at FROM grants WHERE granted_at >= $1 ORDER BY granted_at, grant_id
	`, at)
	if err != nil {
		return nil, fmt.Errorf("failed to read the grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var since []books.Grant
	for rows.Next() {
		var (
			id, country string
			points      int64
			grantedAt   time.Time
		)
		if err := rows.Scan(&id, &country, &points, &grantedAt); err != nil {
			return nil, fmt.Errorf("failed to read a grant: %w", err)
		}
		since = append(since, books.Grant{
			ID: books.GrantID(id), Country: books.Country(country), Points: uint32(points), At: grantedAt.UTC(), //nolint:gosec // CHECK (points > 0), 3 at most.
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the grants: %w", err)
	}
	return since, nil
}
