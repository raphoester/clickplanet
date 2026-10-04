package postgres_worn_title_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ wearing.Store = (*Store)(nil)

func (s *Store) Choice(ctx context.Context, account players.AccountID) (titles.ID, error) {
	var title string
	err := s.db.QueryRowContext(ctx, `SELECT title FROM worn_titles WHERE account_id = $1`, uuid.UUID(account)).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the title chosen: %w", err)
	}
	return titles.ID(title), nil
}

func (s *Store) Choices(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]titles.ID, error) {
	choices := map[players.AccountID]titles.ID{}
	if len(accounts) == 0 {
		return choices, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := s.db.QueryContext(ctx, `SELECT account_id, title FROM worn_titles WHERE account_id = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles chosen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account uuid.UUID
			title   string
		)
		if err := rows.Scan(&account, &title); err != nil {
			return nil, fmt.Errorf("failed to read a title chosen: %w", err)
		}
		choices[players.AccountID(account)] = titles.ID(title)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the titles chosen: %w", err)
	}
	return choices, nil
}

func (s *Store) Wear(ctx context.Context, account players.AccountID, title titles.ID, at time.Time) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO worn_titles (account_id, title, worn_at) VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET title = excluded.title, worn_at = excluded.worn_at
	`, uuid.UUID(account), string(title), at.UTC()); err != nil {
		return fmt.Errorf("failed to wear the title: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM worn_titles WHERE account_id = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to delete the account's worn title: %w", err)
	}
	return nil
}
