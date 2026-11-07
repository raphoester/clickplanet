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

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ decisions.Storage = (*Store)(nil)

const foreignKeyViolation = "23503"

const insert = `INSERT INTO report_decisions (report_id, outcome, comment, decided_at) VALUES ($1, $2::report_outcome, $3, $4)`

func (s *Store) Save(ctx context.Context, decision decisions.Decision) error {
	_, err := s.write(ctx, insert+`
		ON CONFLICT (report_id) DO UPDATE SET
			outcome = EXCLUDED.outcome, comment = EXCLUDED.comment, decided_at = EXCLUDED.decided_at
	`, decision)
	return err
}

func (s *Store) SaveOnce(ctx context.Context, decision decisions.Decision) (bool, error) {
	written, err := s.write(ctx, insert+` ON CONFLICT (report_id) DO NOTHING`, decision)
	return written > 0, err
}

func (s *Store) write(ctx context.Context, statement string, decision decisions.Decision) (int64, error) {
	kept := decision.TakeSnapshot()
	result, err := s.db.On(ctx).ExecContext(ctx, statement,
		uuid.UUID(kept.Report), string(kept.Outcome), kept.Comment, kept.At.UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation {
		return 0, fmt.Errorf("%w: %s", decisions.ErrUnknownReport, uuid.UUID(kept.Report))
	}
	if err != nil {
		return 0, fmt.Errorf("failed to save a decision: %w", err)
	}
	written, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read whether a decision was saved: %w", err)
	}
	return written, nil
}
