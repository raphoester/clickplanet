package postgres_warning_store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ warnings.Store = (*Store)(nil)

func (s *Store) Warned(ctx context.Context, accounts []string) (map[string]warnings.Warning, error) {
	warned := make(map[string]warnings.Warning, len(accounts))
	if len(accounts) == 0 {
		return warned, nil
	}

	found, err := s.read(ctx,
		`SELECT account::text, warned_at, acknowledged_at FROM warnings WHERE account = ANY($1::text[]::uuid[])`, accounts)
	if err != nil {
		return nil, err
	}
	for _, warning := range found {
		warned[warning.Account] = warning
	}
	return warned, nil
}

func (s *Store) Since(ctx context.Context, from time.Time) ([]warnings.Warning, error) {
	return s.read(ctx, `SELECT account::text, warned_at, acknowledged_at FROM warnings WHERE warned_at >= $1`, from.UTC())
}

func (s *Store) Keep(ctx context.Context, warning warnings.Warning) (bool, error) {
	result, err := s.db.On(ctx).ExecContext(ctx,
		`INSERT INTO warnings (account, warned_at) VALUES ($1, $2) ON CONFLICT (account) DO NOTHING`,
		warning.Account, warning.At.UTC())
	if err != nil {
		return false, fmt.Errorf("failed to keep the warning: %w", err)
	}
	kept, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to keep the warning: %w", err)
	}
	return kept == 1, nil
}

func (s *Store) Acknowledge(ctx context.Context, account string, at time.Time) error {
	if _, err := s.db.On(ctx).ExecContext(ctx,
		`UPDATE warnings SET acknowledged_at = $2 WHERE account = $1 AND acknowledged_at IS NULL`,
		account, at.UTC()); err != nil {
		return fmt.Errorf("failed to keep the acknowledgement: %w", err)
	}
	return nil
}

func (s *Store) read(ctx context.Context, query string, args ...any) ([]warnings.Warning, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the warnings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []warnings.Warning
	for rows.Next() {
		var (
			warning      warnings.Warning
			acknowledged sql.NullTime
		)
		if err := rows.Scan(&warning.Account, &warning.At, &acknowledged); err != nil {
			return nil, fmt.Errorf("failed to scan a warning: %w", err)
		}
		warning.At = warning.At.UTC()
		if acknowledged.Valid {
			warning.AcknowledgedAt = acknowledged.Time.UTC()
		}
		found = append(found, warning)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the warnings: %w", err)
	}
	return found, nil
}
