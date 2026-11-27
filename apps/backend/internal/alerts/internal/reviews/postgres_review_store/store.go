package postgres_review_store

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ reviews.Store = (*Store)(nil)

func (s *Store) Told(ctx context.Context, submission reviews.SubmissionID, cause reviews.Cause) (bool, error) {
	var told bool
	if err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM review_posts WHERE submission_id = $1 AND cause = $2::review_cause)
	`, string(submission), string(cause)).Scan(&told); err != nil {
		return false, fmt.Errorf("failed to read whether the staff were told of %s: %w", submission, err)
	}
	return told, nil
}

func (s *Store) SaveTold(
	ctx context.Context, submission reviews.SubmissionID, cause reviews.Cause, at time.Time,
) error {
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO review_posts (submission_id, cause, sent_at) VALUES ($1, $2::review_cause, $3)
		ON CONFLICT (submission_id, cause) DO UPDATE SET sent_at = EXCLUDED.sent_at
	`, string(submission), string(cause), at.UTC()); err != nil {
		return fmt.Errorf("failed to keep the post of %s: %w", submission, err)
	}
	return nil
}
