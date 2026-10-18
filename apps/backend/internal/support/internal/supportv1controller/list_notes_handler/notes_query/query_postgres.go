package notes_query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
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

type Links interface {
	ViewURL(attachment string) (string, error)
}

func NewPostgresQuery(db cppg.Querier, links Links) *PostgresQuery {
	return &PostgresQuery{db: db, links: links}
}

type PostgresQuery struct {
	db    cppg.Querier
	links Links
}

// diagnostics holds the JSON names of support.v1.PageContext, and the user agent beside them.
const notes = `
	SELECT
		id, kind, COALESCE(account_id::text, ''), text, submitted_at,
		diagnostics, COALESCE(diagnostics->>'userAgent', ''),
		COALESCE((
			SELECT json_agg(json_build_object('id', id, 'contentType', media_type, 'sizeBytes', size_bytes) ORDER BY position)
			FROM attachments
			WHERE note_id = notes.id
		), '[]')
	FROM notes
	WHERE ($1::timestamptz IS NULL OR submitted_at >= $1)
		AND ($2::note_kind IS NULL OR kind = $2)
	ORDER BY submitted_at DESC, id DESC
	LIMIT $3
`

// A key this build does not know is one a newer build wrote: the rest of the note is still read.
var kept = protojson.UnmarshalOptions{DiscardUnknown: true}

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
			note     = &supportv1.Note{Context: &supportv1.PageContext{}}
			stored   string
			said     []byte
			attached []byte
			at       time.Time
		)
		if err := rows.Scan(&note.Id, &stored, &note.AccountId, &note.Text, &at, &said, &note.UserAgent, &attached); err != nil {
			return nil, fmt.Errorf("failed to read a note: %w", err)
		}
		if err := kept.Unmarshal(said, note.Context); err != nil {
			return nil, fmt.Errorf("failed to decode the diagnostics of the note %s: %w", note.GetId(), err)
		}
		if note.Attachments, err = q.attachments(note.GetId(), attached); err != nil {
			return nil, err
		}

		known, ok := kinds[stored]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKind, stored)
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

type attachment struct {
	ID          string `json:"id"`
	ContentType string `json:"contentType"`
	SizeBytes   uint64 `json:"sizeBytes"`
}

func (q *PostgresQuery) attachments(note string, stored []byte) ([]*supportv1.NoteAttachment, error) {
	var files []attachment
	if err := json.Unmarshal(stored, &files); err != nil {
		return nil, fmt.Errorf("failed to decode the attachments of the note %s: %w", note, err)
	}

	var attachments []*supportv1.NoteAttachment
	for _, file := range files {
		link, err := q.links.ViewURL(file.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to make the link to the attachment %s of the note %s: %w", file.ID, note, err)
		}
		attachments = append(attachments, &supportv1.NoteAttachment{
			ContentType: file.ContentType, SizeBytes: file.SizeBytes, Url: link,
		})
	}
	return attachments, nil
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
