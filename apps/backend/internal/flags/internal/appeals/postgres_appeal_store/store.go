package postgres_appeal_store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ appeals.Store = (*Store)(nil)

func (s *Store) Of(ctx context.Context, submission submissions.SubmissionID) ([]appeals.Appeal, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT account_id, reason::text, note, state::text, asked_at, decided_at FROM appeals WHERE submission_id = $1
	`, uuid.UUID(submission))
	if err != nil {
		return nil, fmt.Errorf("failed to read the appeals of %s: %w", submission, err)
	}
	defer func() { _ = rows.Close() }()

	var kept []appeals.Appeal
	for rows.Next() {
		var account uuid.UUID
		var reason, state string
		row := appeals.Snapshot{Submission: submission}
		if err := rows.Scan(&account, &reason, &row.Rejection.Note, &state, &row.AskedAt, &row.DecidedAt); err != nil {
			return nil, fmt.Errorf("failed to read an appeal of %s: %w", submission, err)
		}
		row.Account, row.Rejection.Reason, row.State = catalog.AccountID(account), submissions.Reason(reason),
			appeals.State(state)
		row.AskedAt, row.DecidedAt = row.AskedAt.UTC(), row.DecidedAt.UTC()
		appeal, err := row.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore an appeal of %s: %w", submission, err)
		}
		kept = append(kept, appeal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the appeals of %s: %w", submission, err)
	}
	return kept, nil
}

func (s *Store) Save(ctx context.Context, appeal appeals.Appeal) error {
	row := appeal.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO appeals (submission_id, account_id, reason, note, state, asked_at, decided_at)
		VALUES ($1, $2, $3::rejection_reason, $4, $5::appeal_state, $6, $7)
		ON CONFLICT (submission_id) DO UPDATE SET state = EXCLUDED.state, decided_at = EXCLUDED.decided_at
	`, uuid.UUID(row.Submission), uuid.UUID(row.Account), string(row.Rejection.Reason), row.Rejection.Note,
		string(row.State), row.AskedAt.UTC(), row.DecidedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the appeal of %s: %w", row.Submission, err)
	}
	return nil
}
