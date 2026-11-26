package submissions_query

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// The decided submissions the operator reads, the latest first: the waiting ones are all read.
const decidedRead = 500

// Review is which waiting submissions the operator reads.
type Review string

const (
	// Every one, while the shop sells nothing.
	EveryWaiting Review = "every"
	// Only the ones whose time is paid: the shop sells, and a flag is paid for before its review.
	PaidWaiting Review = "paid"
)

func NewPostgresQuery(db cppg.Conn, messages submissionmessage.Messages, review Review) PostgresQuery {
	return PostgresQuery{db: db, messages: messages, review: review}
}

type PostgresQuery struct {
	db       cppg.Conn
	messages submissionmessage.Messages
	review   Review
}

const columns = `s.id, s.account_id, s.flag, s.name, s.pin_latitude, s.pin_longitude,
	s.picture_hash, s.picture_width, s.picture_height, s.picture_stretch, s.picture_focus_x, s.picture_focus_y,
	s.picture_other_colors, s.picture_held, s.state::text, s.submitted_at, s.decided_at,
	COALESCE(r.reason::text, ''), COALESCE(r.note, ''), ` + paid

// Time held for a submission and neither released nor taken back.
const paid = `EXISTS (SELECT 1 FROM time_held h WHERE h.submission_id = s.id
		AND NOT EXISTS (SELECT 1 FROM time_grants g WHERE g.id = h.grant_id)
		AND NOT EXISTS (SELECT 1 FROM time_takebacks t WHERE t.grant_id = h.grant_id))`

const from = ` FROM submissions s LEFT JOIN rejections r ON r.submission_id = s.id`

// Mine is the account's submissions, the newest first.
func (q PostgresQuery) Mine(ctx context.Context, account catalog.AccountID) (*flagsv1.ListMySubmissionsResponse, error) {
	rows, err := q.read(ctx, `SELECT `+columns+from+` WHERE s.account_id = $1 ORDER BY s.submitted_at DESC, s.id`,
		uuid.UUID(account))
	if err != nil {
		return nil, err
	}
	mine := make([]*flagsv1.Submission, 0, len(rows))
	for _, row := range rows {
		mine = append(mine, q.messages.Submission(row))
	}
	held, closures, err := q.held(ctx, account)
	if err != nil {
		return nil, err
	}
	return &flagsv1.ListMySubmissionsResponse{Submissions: mine, Closures: closures, Held: held}, nil
}

// The flags the account holds, and the note of each one the operator removed.
func (q PostgresQuery) held(ctx context.Context, account catalog.AccountID) ([]string, []*flagsv1.Closure, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT h.flag, COALESCE(c.note, '') FROM holders h
		LEFT JOIN closures c ON c.flag = h.flag
		WHERE h.account_id = $1
		ORDER BY h.flag`, uuid.UUID(account))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the account's flags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var held []string
	var closures []*flagsv1.Closure
	for rows.Next() {
		var flag, note string
		if err := rows.Scan(&flag, &note); err != nil {
			return nil, nil, fmt.Errorf("failed to read a flag of the account's: %w", err)
		}
		held = append(held, flag)
		if note != "" {
			closures = append(closures, &flagsv1.Closure{Flag: flag, Note: note})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read the account's flags: %w", err)
	}
	return held, closures, nil
}

// Kept is every submission waiting, or only the paid ones, the oldest first, then the ones decided last, the latest
// first.
func (q PostgresQuery) Kept(ctx context.Context) (*flagsv1.ListSubmissionsResponse, error) {
	rows, err := q.read(ctx, `
		(SELECT `+columns+from+` WHERE s.state = 'waiting' AND ($2 OR `+paid+`) ORDER BY s.submitted_at, s.id)
		UNION ALL
		(SELECT `+columns+from+` WHERE s.state <> 'waiting' ORDER BY s.decided_at DESC, s.id LIMIT $1)`,
		decidedRead, q.review == EveryWaiting)
	if err != nil {
		return nil, err
	}
	judged, err := q.judgements(ctx, rows)
	if err != nil {
		return nil, err
	}
	kept := make([]*flagsv1.KeptSubmission, 0, len(rows))
	for _, row := range rows {
		message, err := q.messages.Kept(row)
		if err != nil {
			return nil, fmt.Errorf("failed to link the picture of %s: %w", row.ID, err)
		}
		if judgement, ok := judged[row.ID]; ok {
			message.Judgement = submissionmessage.Judgement(judgement)
		}
		kept = append(kept, message)
	}
	return &flagsv1.ListSubmissionsResponse{Submissions: kept}, nil
}

// What the judge ruled on the submissions read, by id.
func (q PostgresQuery) judgements(
	ctx context.Context, rows []submissions.Snapshot,
) (map[submissions.SubmissionID]judgements.Snapshot, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, uuid.UUID(row.ID))
	}
	read, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT submission_id, verdict::text, why, judged_at FROM judgements WHERE submission_id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the judgements: %w", err)
	}
	judged := map[submissions.SubmissionID]judgements.Snapshot{}
	for read.Next() {
		var id uuid.UUID
		var verdict string
		var judgement judgements.Snapshot
		if err := read.Scan(&id, &verdict, &judgement.Why, &judgement.JudgedAt); err != nil {
			_ = read.Close()
			return nil, fmt.Errorf("failed to read a judgement: %w", err)
		}
		judgement.Submission, judgement.Verdict = submissions.SubmissionID(id), judgements.Verdict(verdict)
		judged[judgement.Submission] = judgement
	}
	if err := errors.Join(read.Err(), read.Close()); err != nil {
		return nil, fmt.Errorf("failed to read the judgements: %w", err)
	}
	return judged, nil
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
		var flag, name, state, reason string
		var row submissions.Snapshot
		var picture pictures.Snapshot
		var focusX, focusY float32
		var others uint32
		if err := rows.Scan(&id, &account, &flag, &name, &row.Pin.Latitude, &row.Pin.Longitude,
			&picture.Hash, &picture.Width, &picture.Height, &picture.Fit.Stretch, &focusX, &focusY, &others,
			&row.Held, &state, &row.SubmittedAt, &row.DecidedAt, &reason, &row.Rejection.Note, &row.Paid); err != nil {
			return nil, fmt.Errorf("failed to read a submission: %w", err)
		}
		row.ID, row.Flag, row.Name, row.State = submissions.SubmissionID(id), catalog.FlagID(flag), catalog.Name(name),
			submissions.State(state)
		row.Account = catalog.AccountID(account)
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
