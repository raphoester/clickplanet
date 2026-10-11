package notes_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	DefaultLimit = 100
	MaxLimit     = 500
)

var ErrUnknownKind = errors.New("a stored note has a kind the wire cannot carry")

var kinds = map[string]supportv1.NoteKind{
	"suggestion": supportv1.NoteKind_NOTE_KIND_SUGGESTION,
}

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const notes = `
	SELECT id, kind, account_id, text, submitted_at
	FROM notes
	WHERE $1::timestamptz IS NULL OR submitted_at >= $1
	ORDER BY submitted_at DESC, id DESC
	LIMIT $2
`

// A zero since is every note kept; a zero limit is DefaultLimit, and none passes MaxLimit.
func (q *PostgresQuery) Notes(ctx context.Context, since time.Time, limit uint32) (*supportv1.ListNotesResponse, error) {
	rows, err := q.db.QueryContext(ctx, notes, sql.NullTime{Time: since.UTC(), Valid: !since.IsZero()}, limitOf(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &supportv1.ListNotesResponse{}
	for rows.Next() {
		var (
			note = &supportv1.Note{}
			kind string
			at   time.Time
		)
		if err := rows.Scan(&note.Id, &kind, &note.AccountId, &note.Text, &at); err != nil {
			return nil, fmt.Errorf("failed to read a note: %w", err)
		}

		known, ok := kinds[kind]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
		}
		note.Kind = known
		note.SubmittedAt = timestamppb.New(at)
		answer.Notes = append(answer.Notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the notes: %w", err)
	}
	return answer, nil
}

func limitOf(asked uint32) int {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return int(asked)
	}
}
