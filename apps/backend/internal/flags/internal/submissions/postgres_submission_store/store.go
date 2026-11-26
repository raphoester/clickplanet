package postgres_submission_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ submissions.Store = (*Store)(nil)

const columns = `s.id, s.account_id, s.flag, s.name, s.pin_latitude, s.pin_longitude,
	s.picture_hash, s.picture_width, s.picture_height, s.picture_stretch, s.picture_focus_x, s.picture_focus_y,
	s.picture_other_colors, s.picture_held, s.state::text, s.submitted_at, s.decided_at,
	COALESCE(r.reason::text, ''), COALESCE(r.note, ''), ` + paid

// Time held for a submission and neither released nor taken back.
const paid = `EXISTS (SELECT 1 FROM time_held h WHERE h.submission_id = s.id
		AND NOT EXISTS (SELECT 1 FROM time_grants g WHERE g.id = h.grant_id)
		AND NOT EXISTS (SELECT 1 FROM time_takebacks t WHERE t.grant_id = h.grant_id))`

const from = ` FROM submissions s LEFT JOIN rejections r ON r.submission_id = s.id`

func (s *Store) Submission(ctx context.Context, id submissions.SubmissionID) (submissions.Submission, error) {
	// Locked to the end of the transaction a change reads it in: a payment and the operator never open it twice.
	row := s.db.On(ctx).QueryRowContext(ctx, `SELECT `+columns+from+` WHERE s.id = $1 FOR UPDATE OF s`, uuid.UUID(id))
	submission, err := submissionOf(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return submissions.Submission{}, fmt.Errorf("%s: %w", id, submissions.ErrUnknownSubmission)
	}
	return submission, err
}

func (s *Store) Founded(ctx context.Context, account catalog.AccountID) ([]submissions.Submission, error) {
	return s.read(ctx, `SELECT `+columns+from+` WHERE s.account_id = $1 ORDER BY s.submitted_at DESC, s.id`,
		uuid.UUID(account))
}

func (s *Store) Opened(ctx context.Context, flag catalog.FlagID) ([]submissions.Submission, error) {
	return s.read(ctx, `SELECT `+columns+from+` WHERE s.state = 'opened' AND s.flag = $1 ORDER BY s.id`, flag.String())
}

func (s *Store) RejectedHeld(ctx context.Context) ([]submissions.Submission, error) {
	return s.read(ctx, `SELECT `+columns+from+` WHERE s.state = 'rejected' AND s.picture_held
		ORDER BY s.submitted_at DESC, s.id`)
}

func (s *Store) read(ctx context.Context, query string, args ...any) ([]submissions.Submission, error) {
	rows, err := s.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the submissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var kept []submissions.Submission
	for rows.Next() {
		submission, err := submissionOf(rows.Scan)
		if err != nil {
			return nil, err
		}
		kept = append(kept, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the submissions: %w", err)
	}
	return kept, nil
}

func submissionOf(scan func(dest ...any) error) (submissions.Submission, error) {
	var id uuid.UUID
	var account uuid.UUID
	var flag, name, state, reason string
	var kept submissions.Snapshot
	var picture pictures.Snapshot
	var focusX, focusY float32
	var others uint32
	if err := scan(&id, &account, &flag, &name, &kept.Pin.Latitude, &kept.Pin.Longitude,
		&picture.Hash, &picture.Width, &picture.Height, &picture.Fit.Stretch, &focusX, &focusY,
		&others, &kept.Held, &state, &kept.SubmittedAt, &kept.DecidedAt, &reason, &kept.Rejection.Note,
		&kept.Paid); err != nil {
		return submissions.Submission{}, fmt.Errorf("failed to read a submission: %w", err)
	}
	kept.ID, kept.Name, kept.State = submissions.SubmissionID(id), catalog.Name(name), submissions.State(state)
	kept.Account = catalog.AccountID(account)
	if flag != "" {
		flagID, err := catalog.ParseFlagID(flag)
		if err != nil {
			return submissions.Submission{}, fmt.Errorf("the submission %s: %w", kept.ID, err)
		}
		kept.Flag = flagID
	}
	picture.Fit.FocusX, picture.Fit.FocusY = float64(focusX), float64(focusY)
	kept.Picture, kept.OtherColors = picture, pictures.OtherColors(others)
	kept.Rejection.Reason = submissions.Reason(reason)
	kept.SubmittedAt, kept.DecidedAt = kept.SubmittedAt.UTC(), kept.DecidedAt.UTC()
	submission, err := kept.Restore()
	if err != nil {
		return submissions.Submission{}, fmt.Errorf("failed to restore a submission kept: %w", err)
	}
	return submission, nil
}

func (s *Store) Save(ctx context.Context, kept ...submissions.Submission) error {
	for _, submission := range kept {
		row := submission.TakeSnapshot()
		picture := row.Picture
		if _, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO submissions (id, account_id, flag, name, pin_latitude, pin_longitude,
				picture_hash, picture_width, picture_height, picture_stretch, picture_focus_x, picture_focus_y,
				picture_other_colors, picture_held, state, submitted_at, decided_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::submission_state, $16, $17)
			ON CONFLICT (id) DO UPDATE SET flag = EXCLUDED.flag,
				name = EXCLUDED.name, pin_latitude = EXCLUDED.pin_latitude, pin_longitude = EXCLUDED.pin_longitude,
				picture_held = EXCLUDED.picture_held, state = EXCLUDED.state, decided_at = EXCLUDED.decided_at
		`, uuid.UUID(row.ID), uuid.UUID(row.Account), row.Flag.String(), row.Name.String(), row.Pin.Latitude,
			row.Pin.Longitude, picture.Hash, picture.Width, picture.Height, picture.Fit.Stretch,
			float32(picture.Fit.FocusX), float32(picture.Fit.FocusY), uint32(row.OtherColors), row.Held,
			string(row.State), row.SubmittedAt.UTC(), row.DecidedAt.UTC()); err != nil {
			return fmt.Errorf("failed to keep the submission %s: %w", row.ID, err)
		}
		if row.State != submissions.Rejected {
			continue
		}
		if _, err := s.db.On(ctx).ExecContext(ctx, `
			INSERT INTO rejections (submission_id, reason, note) VALUES ($1, $2::rejection_reason, $3)
			ON CONFLICT (submission_id) DO UPDATE SET reason = EXCLUDED.reason, note = EXCLUDED.note
		`, uuid.UUID(row.ID), string(row.Rejection.Reason), row.Rejection.Note); err != nil {
			return fmt.Errorf("failed to keep the rejection of %s: %w", row.ID, err)
		}
	}
	return nil
}

func (s *Store) DeleteWaiting(ctx context.Context, id submissions.SubmissionID) error {
	result, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM submissions WHERE id = $1 AND state = 'waiting'`,
		uuid.UUID(id))
	if err != nil {
		return fmt.Errorf("failed to delete the submission %s: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read whether the submission %s was deleted: %w", id, err)
	}
	if deleted == 0 {
		return fmt.Errorf("%w: %s", submissions.ErrNotWaiting, id)
	}
	return nil
}
