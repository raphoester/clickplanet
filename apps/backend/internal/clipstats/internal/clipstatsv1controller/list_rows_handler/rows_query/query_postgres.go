package rows_query

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	clipstatsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	DefaultLimit = 500
	MaxLimit     = 2000
)

var ErrInvalidPage = errors.New("not a page this query gave")

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

// After the last row of the page before, in the order of the primary key.
const rows = `
	SELECT taken_at, post_id, platform, document::text
	FROM snapshot_rows
	WHERE ($1::timestamptz IS NULL OR taken_at >= $1)
		AND ($2::timestamptz IS NULL OR taken_at < $2)
		AND ($3::timestamptz IS NULL OR (taken_at, post_id, platform) > ($3, $4, $5))
	ORDER BY taken_at, post_id, platform
	LIMIT $6
`

// Where a page ends: the key of its last row.
type cursor struct {
	TakenAt  time.Time `json:"t"`
	PostID   string    `json:"p"`
	Platform string    `json:"f"`
}

// A zero since or until is no bound; a zero limit is DefaultLimit, and none passes MaxLimit. An empty page is the first.
func (q *PostgresQuery) Rows(ctx context.Context, since, until time.Time, limit uint32, page string) (*clipstatsv1.ListRowsResponse, error) {
	after, err := cursorOf(page)
	if err != nil {
		return nil, err
	}
	size := limitOf(limit)

	// One more than the page: whether there is a next one.
	read, err := q.db.QueryContext(ctx, rows, bound(since), bound(until),
		bound(after.TakenAt), after.PostID, after.Platform, size+1)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rows: %w", err)
	}
	defer func() { _ = read.Close() }()

	listed := &clipstatsv1.ListRowsResponse{}
	for read.Next() {
		var takenAt time.Time
		row := &clipstatsv1.Row{}
		if err := read.Scan(&takenAt, &row.PostId, &row.Platform, &row.Document); err != nil {
			return nil, fmt.Errorf("failed to read a row: %w", err)
		}
		row.TakenAt = timestamppb.New(takenAt)
		listed.Rows = append(listed.Rows, row)
	}
	if err := read.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rows: %w", err)
	}

	if len(listed.Rows) > int(size) {
		listed.Rows = listed.Rows[:size]
		last := listed.Rows[size-1]
		listed.NextPage = pageOf(cursor{TakenAt: last.GetTakenAt().AsTime(), PostID: last.GetPostId(), Platform: last.GetPlatform()})
	}
	return listed, nil
}

func cursorOf(page string) (cursor, error) {
	if page == "" {
		return cursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(page)
	if err != nil {
		return cursor{}, ErrInvalidPage
	}
	var after cursor
	if json.Unmarshal(raw, &after) != nil || after.TakenAt.IsZero() {
		return cursor{}, ErrInvalidPage
	}
	return after, nil
}

func pageOf(after cursor) string {
	raw, _ := json.Marshal(after) //nolint:errchkjson // a time and two strings.
	return base64.RawURLEncoding.EncodeToString(raw)
}

func bound(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
}

func limitOf(asked uint32) uint32 {
	if asked == 0 {
		return DefaultLimit
	}
	return min(asked, MaxLimit)
}
