package postgres_title_store

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

func (s *Store) Holdings(ctx context.Context, accounts []players.AccountID) (titles.Holdings, error) {
	holdings := titles.Holdings{}
	if len(accounts) == 0 {
		return holdings, nil
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, title FROM titles WHERE account_id = ANY($1::uuid[]) ORDER BY account_id, earned_at, title
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles of a page: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account uuid.UUID
			id      string
		)
		if err := rows.Scan(&account, &id); err != nil {
			return nil, fmt.Errorf("failed to read a title: %w", err)
		}
		holdings[players.AccountID(account)] = append(holdings[players.AccountID(account)], titles.ID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the titles of a page: %w", err)
	}
	return holdings, nil
}

func (s *Store) Grant(ctx context.Context, grants titles.Holdings, at time.Time) error {
	accounts, granted := pairsOf(grants)
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

func (s *Store) Revoke(ctx context.Context, revocations titles.Holdings) error {
	accounts, revoked := pairsOf(revocations)
	if len(revoked) == 0 {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM titles USING unnest($1::uuid[], $2::text[]) AS revoked (account_id, title)
		WHERE titles.account_id = revoked.account_id AND titles.title = revoked.title
	`, pq.Array(accounts), pq.Array(revoked)); err != nil {
		return fmt.Errorf("failed to revoke the titles: %w", err)
	}
	return nil
}

func (s *Store) Worn(ctx context.Context, account players.AccountID) (titles.ID, error) {
	var title string
	err := s.db.QueryRowContext(ctx, `SELECT title FROM worn_titles WHERE account_id = $1`, uuid.UUID(account)).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the worn title: %w", err)
	}
	return titles.ID(title), nil
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

func (s *Store) DeleteAccount(ctx context.Context, account players.AccountID) (err error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin deleting the account's titles: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, statement := range []string{
		`DELETE FROM titles WHERE account_id = $1`,
		`DELETE FROM worn_titles WHERE account_id = $1`,
	} {
		if _, err := tx.ExecContext(ctx, statement, uuid.UUID(account)); err != nil {
			return fmt.Errorf("failed to delete the account's titles: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the deletion of the account's titles: %w", err)
	}
	return nil
}

func pairsOf(holdings titles.Holdings) (accounts, ids []string) {
	for account, held := range holdings {
		for _, id := range held {
			accounts = append(accounts, account.String())
			ids = append(ids, string(id))
		}
	}
	return accounts, ids
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
