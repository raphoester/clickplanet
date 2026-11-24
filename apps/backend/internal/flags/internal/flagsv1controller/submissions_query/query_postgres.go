package submissions_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// The decided submissions the operator reads, the latest first: the waiting ones are all read.
const decidedRead = 500

func NewPostgresQuery(db cppg.Conn, messages submissionmessage.Messages) PostgresQuery {
	return PostgresQuery{db: db, messages: messages}
}

type PostgresQuery struct {
	db       cppg.Conn
	messages submissionmessage.Messages
}

const columns = `s.id, s.account_id, s.kind::text, s.flag, s.name, s.pin_latitude, s.pin_longitude,
	s.picture_hash, s.picture_width, s.picture_height, s.picture_stretch, s.picture_focus_x, s.picture_focus_y,
	s.picture_other_colors, s.picture_held, s.state::text, s.submitted_at, s.decided_at,
	COALESCE(r.reason::text, ''), COALESCE(r.note, '')`

const from = ` FROM submissions s LEFT JOIN rejections r ON r.submission_id = s.id`

// Mine is the account's submissions, the newest first.
func (q PostgresQuery) Mine(ctx context.Context, account submissions.AccountID) (*flagsv1.ListMySubmissionsResponse, error) {
	rows, err := q.read(ctx, `SELECT `+columns+from+` WHERE s.account_id = $1 ORDER BY s.submitted_at DESC, s.id`,
		uuid.UUID(account))
	if err != nil {
		return nil, err
	}
	mine := make([]*flagsv1.Submission, 0, len(rows))
	for _, row := range rows {
		mine = append(mine, q.messages.Submission(row))
	}
	closures, err := q.closures(ctx, account)
	if err != nil {
		return nil, err
	}
	return &flagsv1.ListMySubmissionsResponse{Submissions: mine, Closures: closures}, nil
}

// The notes of the flags the account founded: each opened from one of its submissions.
func (q PostgresQuery) closures(ctx context.Context, account submissions.AccountID) ([]*flagsv1.Closure, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT c.flag, c.note FROM closures c
		JOIN submissions s ON s.flag = c.flag AND s.state = 'opened'
		WHERE s.account_id = $1
		ORDER BY c.flag`, uuid.UUID(account))
	if err != nil {
		return nil, fmt.Errorf("failed to read the closes of the account's flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var closures []*flagsv1.Closure
	for rows.Next() {
		closure := &flagsv1.Closure{}
		if err := rows.Scan(&closure.Flag, &closure.Note); err != nil {
			return nil, fmt.Errorf("failed to read a close of the account's flags: %w", err)
		}
		closures = append(closures, closure)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the closes of the account's flags: %w", err)
	}
	return closures, nil
}

// Kept is every submission waiting, the oldest first, then the ones decided last, the latest first.
func (q PostgresQuery) Kept(ctx context.Context) (*flagsv1.ListSubmissionsResponse, error) {
	rows, err := q.read(ctx, `
		(SELECT `+columns+from+` WHERE s.state = 'waiting' ORDER BY s.submitted_at, s.id)
		UNION ALL
		(SELECT `+columns+from+` WHERE s.state <> 'waiting' ORDER BY s.decided_at DESC, s.id LIMIT $1)`,
		decidedRead)
	if err != nil {
		return nil, err
	}
	kept := make([]*flagsv1.KeptSubmission, 0, len(rows))
	for _, row := range rows {
		message, err := q.messages.Kept(row)
		if err != nil {
			return nil, fmt.Errorf("failed to link the picture of %s: %w", row.ID, err)
		}
		kept = append(kept, message)
	}
	return &flagsv1.ListSubmissionsResponse{Submissions: kept}, nil
}

func (q PostgresQuery) read(ctx context.Context, query string, args ...any) ([]submissions.Snapshot, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the submissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var read []submissions.Snapshot
	for rows.Next() {
		var id uuid.UUID
		var account uuid.UUID
		var kind, flag, name, state, reason string
		var row submissions.Snapshot
		var picture pictures.Snapshot
		var focusX, focusY float32
		var others uint32
		if err := rows.Scan(&id, &account, &kind, &flag, &name, &row.Pin.Latitude, &row.Pin.Longitude,
			&picture.Hash, &picture.Width, &picture.Height, &picture.Fit.Stretch, &focusX, &focusY, &others,
			&row.Held, &state, &row.SubmittedAt, &row.DecidedAt, &reason, &row.Rejection.Note); err != nil {
			return nil, fmt.Errorf("failed to read a submission: %w", err)
		}
		row.ID, row.Kind, row.Flag, row.Name, row.State = submissions.SubmissionID(id), submissions.Kind(kind),
			catalog.FlagID(flag), catalog.Name(name), submissions.State(state)
		row.Account = submissions.AccountID(account)
		picture.Fit.FocusX, picture.Fit.FocusY = float64(focusX), float64(focusY)
		row.Picture, row.OtherColors, row.Rejection.Reason = picture, pictures.OtherColors(others),
			submissions.Reason(reason)
		read = append(read, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the submissions: %w", err)
	}
	return read, nil
}
