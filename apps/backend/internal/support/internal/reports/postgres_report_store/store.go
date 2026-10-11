package postgres_report_store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

func New(db cppg.QuerierBeginner) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.QuerierBeginner
}

var _ reports.Storage = (*Store)(nil)

func (s *Store) Save(ctx context.Context, saved ...reports.Report) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin saving reports: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, report := range saved {
		if err := write(ctx, tx, report.TakeSnapshot()); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit the reports saved: %w", err)
	}
	return nil
}

func write(ctx context.Context, tx cppg.Querier, kept reports.Snapshot) error {
	if kept.Forgotten {
		if _, err := tx.ExecContext(ctx, `DELETE FROM reports WHERE id = $1`, uuid.UUID(kept.ID)); err != nil {
			return fmt.Errorf("failed to delete a forgotten report: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reports (id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			reporter_account_id = EXCLUDED.reporter_account_id, reported_account_id = EXCLUDED.reported_account_id,
			reason = EXCLUDED.reason, text = EXCLUDED.text, message_id = EXCLUDED.message_id,
			reported_at = EXCLUDED.reported_at
	`,
		uuid.UUID(kept.ID), uuid.UUID(kept.Reporter), uuid.UUID(kept.Reported),
		string(kept.Complaint.Reason), kept.Complaint.Text, string(kept.Complaint.Message), kept.At.UTC(),
	); err != nil {
		return fmt.Errorf("failed to upsert a report: %w", err)
	}
	return nil
}

func (s *Store) FiledBy(ctx context.Context, reporter reports.AccountID, about reports.AccountID) ([]reports.Report, error) {
	return s.restored(ctx, `
		SELECT id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at
		FROM reports
		WHERE reporter_account_id = $1 AND reported_account_id = $2
		ORDER BY reported_at, id
	`, uuid.UUID(reporter), uuid.UUID(about))
}

func (s *Store) Naming(ctx context.Context, account reports.AccountID) ([]reports.Report, error) {
	return s.restored(ctx, `
		SELECT id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at
		FROM reports
		WHERE reporter_account_id = $1 OR reported_account_id = $1
		ORDER BY reported_at, id
	`, uuid.UUID(account))
}

func (s *Store) FiledBefore(ctx context.Context, cutoff time.Time, limit int) ([]reports.Report, error) {
	return s.restored(ctx, `
		SELECT id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at
		FROM reports
		WHERE reported_at < $1
		ORDER BY reported_at, id
		LIMIT $2
	`, cutoff.UTC(), limit)
}

func (s *Store) restored(ctx context.Context, query string, args ...any) ([]reports.Report, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var found []reports.Report
	for rows.Next() {
		var (
			id, reporter, reported uuid.UUID
			reason, text, message  string
			at                     time.Time
		)
		if err := rows.Scan(&id, &reporter, &reported, &reason, &text, &message, &at); err != nil {
			return nil, fmt.Errorf("failed to read a report: %w", err)
		}

		report, err := reports.Snapshot{
			ID: reports.ReportID(id), Reporter: reports.AccountID(reporter), Reported: reports.AccountID(reported), At: at.UTC(),
			Complaint: reports.Complaint{Reason: reports.Reason(reason), Text: text, Message: reports.MessageID(message)},
		}.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the report %s: %w", id, err)
		}
		found = append(found, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read reports: %w", err)
	}
	return found, nil
}
