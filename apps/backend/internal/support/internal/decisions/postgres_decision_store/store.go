package postgres_decision_store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

// The decisions on the reports of players, in report_decisions.
func ForReports(db cppg.Querier) *Store[reports.ReportID] {
	return &Store[reports.ReportID]{
		db: db, upsert: upsertInto("report_decisions"),
		id: func(report reports.ReportID) uuid.UUID { return uuid.UUID(report) },
	}
}

// The decisions on the reports of places, in place_report_decisions.
func ForPlaces(db cppg.Querier) *Store[places.ReportID] {
	return &Store[places.ReportID]{
		db: db, upsert: upsertInto("place_report_decisions"),
		id: func(report places.ReportID) uuid.UUID { return uuid.UUID(report) },
	}
}

type Store[R comparable] struct {
	db     cppg.Querier
	upsert string
	id     func(R) uuid.UUID
}

var (
	_ decisions.Storage[reports.ReportID] = (*Store[reports.ReportID])(nil)
	_ decisions.Storage[places.ReportID]  = (*Store[places.ReportID])(nil)
)

const foreignKeyViolation = "23503"

func upsertInto(table string) string {
	return `
		INSERT INTO ` + table + ` (report_id, outcome, comment, decided_at) VALUES ($1, $2::report_outcome, $3, $4)
		ON CONFLICT (report_id) DO UPDATE SET
			outcome = EXCLUDED.outcome, comment = EXCLUDED.comment, decided_at = EXCLUDED.decided_at
	`
}

func (s *Store[R]) Save(ctx context.Context, decision decisions.Decision[R]) error {
	kept := decision.TakeSnapshot()
	_, err := s.db.ExecContext(ctx, s.upsert, s.id(kept.Report), string(kept.Outcome), kept.Comment, kept.At.UTC())

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == foreignKeyViolation {
		return fmt.Errorf("%w: %s", decisions.ErrUnknownReport, s.id(kept.Report))
	}
	if err != nil {
		return fmt.Errorf("failed to save a decision: %w", err)
	}
	return nil
}
