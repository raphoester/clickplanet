package arrivals_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrUnknownSource = errors.New("an arrival source this build does not know")

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

const counts = `
	SELECT to_char(arrived_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, source::text, count(*)
	FROM arrivals
	WHERE ($1::timestamptz IS NULL OR arrived_at >= $1)
		AND ($2::timestamptz IS NULL OR arrived_at < $2)
	GROUP BY day, source
	ORDER BY day, source::text
`

var sources = map[string]authv1.ArrivalSource{
	"unknown":       authv1.ArrivalSource_ARRIVAL_SOURCE_UNKNOWN,
	"youtube":       authv1.ArrivalSource_ARRIVAL_SOURCE_YOUTUBE,
	"tiktok":        authv1.ArrivalSource_ARRIVAL_SOURCE_TIKTOK,
	"recruit":       authv1.ArrivalSource_ARRIVAL_SOURCE_RECRUIT,
	"recruit_video": authv1.ArrivalSource_ARRIVAL_SOURCE_RECRUIT_VIDEO,
	"crazygames":    authv1.ArrivalSource_ARRIVAL_SOURCE_CRAZYGAMES,
}

// A zero since or until is no bound.
func (q *PostgresQuery) Counts(ctx context.Context, since, until time.Time) (*authv1.CountArrivalsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, counts, bound(since), bound(until))
	if err != nil {
		return nil, fmt.Errorf("failed to count the arrivals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counted := &authv1.CountArrivalsResponse{}
	for rows.Next() {
		var (
			day      string
			source   string
			accounts uint32
		)
		if err := rows.Scan(&day, &source, &accounts); err != nil {
			return nil, fmt.Errorf("failed to read a count of arrivals: %w", err)
		}
		known, ok := sources[source]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownSource, source)
		}
		counted.Counts = append(counted.Counts, &authv1.ArrivalCount{Day: day, Source: known, Accounts: accounts})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the counts of arrivals: %w", err)
	}
	return counted, nil
}

func bound(at time.Time) sql.NullTime {
	return sql.NullTime{Time: at.UTC(), Valid: !at.IsZero()}
}
