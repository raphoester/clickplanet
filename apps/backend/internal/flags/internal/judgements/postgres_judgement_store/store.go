package postgres_judgement_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ judgements.Store = (*Store)(nil)

func (s *Store) Unjudged(ctx context.Context, limit int) ([]submissions.SubmissionID, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, `
		SELECT s.id FROM submissions s
		WHERE s.state = 'waiting' AND NOT EXISTS (SELECT 1 FROM judgements j WHERE j.submission_id = s.id)
		ORDER BY s.submitted_at, s.id LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list the submissions to judge: %w", err)
	}
	var unjudged []submissions.SubmissionID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to read a submission to judge: %w", err)
		}
		unjudged = append(unjudged, submissions.SubmissionID(id))
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("failed to list the submissions to judge: %w", err)
	}
	return unjudged, nil
}

func (s *Store) Judgement(ctx context.Context, submission submissions.SubmissionID) (judgements.Judgement, error) {
	var verdict string
	kept := judgements.Snapshot{Submission: submission}
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT verdict::text, why, judged_at FROM judgements WHERE submission_id = $1
	`, uuid.UUID(submission)).Scan(&verdict, &kept.Why, &kept.JudgedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return judgements.Judgement{}, fmt.Errorf("%w: %s", judgements.ErrUnjudged, submission)
	}
	if err != nil {
		return judgements.Judgement{}, fmt.Errorf("failed to read the judgement of %s: %w", submission, err)
	}
	kept.Verdict, kept.JudgedAt = judgements.Verdict(verdict), kept.JudgedAt.UTC()
	judgement, err := kept.Restore()
	if err != nil {
		return judgements.Judgement{}, fmt.Errorf("failed to restore the judgement of %s: %w", submission, err)
	}
	return judgement, nil
}

func (s *Store) Save(ctx context.Context, judgement judgements.Judgement) error {
	kept := judgement.TakeSnapshot()
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO judgements (submission_id, verdict, why, judged_at) VALUES ($1, $2::judgement_verdict, $3, $4)
		ON CONFLICT (submission_id) DO NOTHING
	`, uuid.UUID(kept.Submission), string(kept.Verdict), kept.Why, kept.JudgedAt.UTC()); err != nil {
		return fmt.Errorf("failed to keep the judgement of %s: %w", kept.Submission, err)
	}
	return nil
}
