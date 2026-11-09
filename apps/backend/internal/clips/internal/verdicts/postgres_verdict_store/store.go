package postgres_verdict_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

// A verdict kept again for an account replaces the one it had.
func (s *Store) Keep(ctx context.Context, kept []verdicts.Verdict) error {
	for _, verdict := range kept {
		snapshot := verdict.TakeSnapshot()
		if _, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO name_verdicts (account, name, shown, reason, decided_by, decided_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (account) DO UPDATE SET
				name = EXCLUDED.name, shown = EXCLUDED.shown, reason = EXCLUDED.reason,
				decided_by = EXCLUDED.decided_by, decided_at = EXCLUDED.decided_at
		`, uuid.UUID(snapshot.Account), snapshot.Name, snapshot.Shown, snapshot.Reason, string(snapshot.By), snapshot.At); err != nil {
			return fmt.Errorf("failed to keep the verdict of a name: %w", err)
		}
	}
	return nil
}

func (s *Store) Of(ctx context.Context, accounts []verdicts.AccountID) ([]verdicts.Verdict, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.String())
	}
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT account, name, shown, reason, decided_by, decided_at
		FROM name_verdicts WHERE account = ANY($1::uuid[])
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the verdicts of the names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []verdicts.Verdict
	for rows.Next() {
		verdict, err := scan(rows)
		if err != nil {
			return nil, err
		}
		found = append(found, verdict)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the verdicts of the names: %w", err)
	}
	return found, nil
}

func (s *Store) Verdict(ctx context.Context, account verdicts.AccountID) (verdicts.Verdict, error) {
	verdict, err := scan(s.db.On(ctx).QueryRowContext(ctx, `
		SELECT account, name, shown, reason, decided_by, decided_at FROM name_verdicts WHERE account = $1
	`, uuid.UUID(account)))
	if errors.Is(err, sql.ErrNoRows) {
		return verdicts.Verdict{}, verdicts.ErrNoVerdict
	}
	return verdict, err
}

func (s *Store) Forget(ctx context.Context, account verdicts.AccountID) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM name_verdicts WHERE account = $1`, uuid.UUID(account)); err != nil {
		return fmt.Errorf("failed to forget the verdict of the account's name: %w", err)
	}
	return nil
}

type row interface {
	Scan(dest ...any) error
}

func scan(r row) (verdicts.Verdict, error) {
	var (
		account uuid.UUID
		by      string
		kept    verdicts.Snapshot
	)
	if err := r.Scan(&account, &kept.Name, &kept.Shown, &kept.Reason, &by, &kept.At); err != nil {
		return verdicts.Verdict{}, fmt.Errorf("failed to read the verdict of a name: %w", err)
	}
	decider, err := verdicts.DeciderOf(by)
	if err != nil {
		return verdicts.Verdict{}, err //nolint:wrapcheck // the domain named what is wrong.
	}
	kept.Account, kept.By, kept.At = verdicts.AccountID(account), decider, kept.At.UTC()
	return kept.Restore(), nil
}
