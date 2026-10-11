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

var (
	ErrUnknownKind = errors.New("a stored note has a kind the wire cannot carry")
	ErrInvalidKind = errors.New("not a kind of note")
)

var kinds = map[string]supportv1.NoteKind{
	"suggestion": supportv1.NoteKind_NOTE_KIND_SUGGESTION,
	"bug":        supportv1.NoteKind_NOTE_KIND_BUG,
}

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const notes = `
	SELECT
		id, kind, account_id, text, submitted_at,
		diagnostics IS NOT NULL,
		COALESCE(diagnostics->>'build', ''),
		COALESCE((diagnostics->>'viewportWidth')::bigint, 0),
		COALESCE((diagnostics->>'viewportHeight')::bigint, 0),
		COALESCE((diagnostics->>'connectionDown')::boolean, false),
		COALESCE(diagnostics->>'userAgent', '')
	FROM notes
	WHERE ($1::timestamptz IS NULL OR submitted_at >= $1)
		AND ($2::note_kind IS NULL OR kind = $2)
	ORDER BY submitted_at DESC, id DESC
	LIMIT $3
`

// A zero since is every note kept, and an unspecified kind every kind; a zero limit is DefaultLimit, and none
// passes MaxLimit.
func (q *PostgresQuery) Notes(
	ctx context.Context, since time.Time, kind supportv1.NoteKind, limit uint32,
) (*supportv1.ListNotesResponse, error) {
	asked, err := storedKind(kind)
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, notes,
		sql.NullTime{Time: since.UTC(), Valid: !since.IsZero()}, asked, limitOf(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &supportv1.ListNotesResponse{}
	for rows.Next() {
		var (
			note      = &supportv1.Note{}
			page      = &supportv1.BugContext{}
			stored    string
			at        time.Time
			diagnosed bool
		)
		if err := rows.Scan(&note.Id, &stored, &note.AccountId, &note.Text, &at, &diagnosed,
			&page.Build, &page.ViewportWidth, &page.ViewportHeight, &page.ConnectionDown, &note.UserAgent); err != nil {
			return nil, fmt.Errorf("failed to read a note: %w", err)
		}

		known, ok := kinds[stored]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKind, stored)
		}
		note.Kind = known
		note.SubmittedAt = timestamppb.New(at)
		if diagnosed {
			note.Context = page
		}
		answer.Notes = append(answer.Notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the notes: %w", err)
	}
	return answer, nil
}

func storedKind(asked supportv1.NoteKind) (sql.NullString, error) {
	if asked == supportv1.NoteKind_NOTE_KIND_UNSPECIFIED {
		return sql.NullString{}, nil
	}
	for stored, kind := range kinds {
		if kind == asked {
			return sql.NullString{String: stored, Valid: true}, nil
		}
	}
	return sql.NullString{}, fmt.Errorf("%w: %d", ErrInvalidKind, asked)
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
