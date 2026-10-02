package postgres_title_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ titles.Store = (*Store)(nil)

func (s *Store) Held(ctx context.Context, account players.AccountID) (titles.IDs, error) {
	return idsOf(s.db.QueryContext(ctx, `SELECT title FROM titles WHERE account_id = $1 ORDER BY earned_at, title`, uuid.UUID(account)))
}

func (s *Store) Grant(ctx context.Context, grants titles.Grants, at time.Time) error {
	var accounts, granted []string
	for account, ids := range grants {
		for _, id := range ids {
			accounts = append(accounts, account.String())
			granted = append(granted, string(id))
		}
	}
	if len(granted) == 0 {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO titles (account_id, title, earned_at)
		SELECT granted.account_id, granted.title, $3
		FROM unnest($1::uuid[], $2::text[]) AS granted (account_id, title)
		ON CONFLICT (account_id, title) DO NOTHING
	`, pq.Array(accounts), pq.Array(granted), at.UTC()); err != nil {
		return fmt.Errorf("failed to grant the titles: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM titles WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's titles: %w", err)
	}
	return nil
}

func (s *Store) Backfilled(ctx context.Context) (titles.IDs, error) {
	return idsOf(s.db.QueryContext(ctx, `SELECT title FROM title_backfills ORDER BY title`))
}

func (s *Store) SaveBackfilled(ctx context.Context, ids titles.IDs, at time.Time) error {
	backfilled := make([]string, len(ids))
	for i, id := range ids {
		backfilled[i] = string(id)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO title_backfills (title, backfilled_at) SELECT unnest($1::text[]), $2
		ON CONFLICT (title) DO NOTHING
	`, pq.Array(backfilled), at.UTC()); err != nil {
		return fmt.Errorf("failed to save the backfilled titles: %w", err)
	}
	return nil
}

func idsOf(rows *sql.Rows, err error) (titles.IDs, error) {
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids titles.IDs
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to read a title: %w", err)
		}
		ids = append(ids, titles.ID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return ids, nil
}
