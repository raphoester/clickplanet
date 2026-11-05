package postgres_optout_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

func (s *Store) Keep(ctx context.Context, account optouts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO opt_outs (account) VALUES ($1) ON CONFLICT DO NOTHING
	`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to keep the player out of the clips: %w", err)
	}
	return nil
}

func (s *Store) Forget(ctx context.Context, account optouts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM opt_outs WHERE account = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the player's opt-out: %w", err)
	}
	return nil
}

func (s *Store) Among(ctx context.Context, accounts []optouts.AccountID) ([]optouts.AccountID, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}

	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT account FROM opt_outs WHERE account = ANY($1::uuid[])`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read who keeps out of the clips: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []optouts.AccountID
	for rows.Next() {
		var account uuid.UUID
		if err := rows.Scan(&account); err != nil {
			return nil, fmt.Errorf("failed to read an opt-out: %w", err)
		}
		out = append(out, optouts.AccountID(account))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read who keeps out of the clips: %w", err)
	}
	return out, nil
}
