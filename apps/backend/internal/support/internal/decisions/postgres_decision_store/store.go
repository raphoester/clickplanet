package postgres_decision_store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
)

func New(db cppg.Querier) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Querier
}

var _ decisions.Storage = (*Store)(nil)

const foreignKeyViolation = "23503"

func (s *Store) Save(ctx context.Context, decision decisions.Decision) error {
	kept := decision.TakeSnapshot()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO report_decisions (report_id, outcome, comment, decided_at) VALUES ($1, $2::report_outcome, $3, $4)
		ON CONFLICT (report_id) DO UPDATE SET
			outcome = EXCLUDED.outcome, comment = EXCLUDED.comment, decided_at = EXCLUDED.decided_at
	`, uuid.UUID(kept.Report), string(kept.Outcome), kept.Comment, kept.At.UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation {
		return fmt.Errorf("%w: %s", decisions.ErrUnknownReport, uuid.UUID(kept.Report))
	}
	if err != nil {
		return fmt.Errorf("failed to save a decision: %w", err)
	}
	return nil
}
