package postgres_rule_change_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ rules.Store = (*Store)(nil)

func (s *Store) Change(ctx context.Context, request rules.RequestID) (rules.Change, error) {
	return keptOf(s.db.QueryRowContext(ctx, `
		SELECT request_id, rule_set, source, applied_at FROM rule_changes WHERE request_id = $1
	`, string(request)))
}

func (s *Store) Latest(ctx context.Context) (rules.Change, error) {
	return keptOf(s.db.QueryRowContext(ctx, `
		SELECT request_id, rule_set, source, applied_at FROM rule_changes ORDER BY applied_at DESC, seq DESC LIMIT 1
	`))
}

func (s *Store) Save(ctx context.Context, change rules.Change) error {
	kept := change.TakeSnapshot()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO rule_changes (request_id, rule_set, source, applied_at) VALUES ($1, $2, $3::rule_change_source, $4)
		ON CONFLICT (request_id) DO UPDATE
		SET rule_set = EXCLUDED.rule_set, source = EXCLUDED.source, applied_at = EXCLUDED.applied_at
	`, string(kept.Request), string(kept.Set), string(kept.Source), kept.AppliedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the rule change %q: %w", kept.Request, err)
	}

	return nil
}

func keptOf(row *sql.Row) (rules.Change, error) {
	var (
		request, set, source string
		appliedAt            time.Time
	)
	err := row.Scan(&request, &set, &source, &appliedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return rules.Change{}, nil
	case err != nil:
		return rules.Change{}, fmt.Errorf("failed to read a rule change: %w", err)
	}

	return rules.Snapshot{
		Request:   rules.RequestID(request),
		Set:       rules.SetName(set),
		Source:    rules.Source(source),
		AppliedAt: appliedAt.UTC(),
	}.Restore(), nil
}
