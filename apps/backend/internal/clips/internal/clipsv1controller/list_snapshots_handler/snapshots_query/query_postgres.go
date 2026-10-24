package snapshots_query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const (
	DefaultLimit = 100
	MaxLimit     = 500
)

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

const snapshots = `
	SELECT summary
	FROM snapshots
	WHERE ($1::timestamptz IS NULL OR taken_at >= $1)
		AND ($2::timestamptz IS NULL OR taken_at < $2)
	ORDER BY taken_at DESC
	LIMIT $3
`

// A key this build does not know is one a newer job wrote: the rest of the summary is still read.
var kept = protojson.UnmarshalOptions{DiscardUnknown: true}

// A zero since or until is no bound; a zero limit is DefaultLimit, and none passes MaxLimit.
func (q *PostgresQuery) Snapshots(ctx context.Context, since, until time.Time, limit uint32) (*clipsv1.ListSnapshotsResponse, error) {
	rows, err := q.db.QueryContext(ctx, snapshots, bound(since), bound(until), limitOf(limit, DefaultLimit, MaxLimit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	listed := &clipsv1.ListSnapshotsResponse{}
	for rows.Next() {
		var summary []byte
		if err := rows.Scan(&summary); err != nil {
			return nil, fmt.Errorf("failed to read a snapshot: %w", err)
		}
		snapshot := &clipsv1.StatsSnapshot{}
		if err := kept.Unmarshal(summary, snapshot); err != nil {
			return nil, fmt.Errorf("a kept summary is not a clips.v1.StatsSnapshot: %w", err)
		}
		listed.Snapshots = append(listed.Snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the snapshots: %w", err)
	}
	return listed, nil
}

func bound(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
}

func limitOf(asked, byDefault, most uint32) uint32 {
	if asked == 0 {
		return byDefault
	}
	return min(asked, most)
}
