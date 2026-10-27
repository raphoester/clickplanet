package seasons_query

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// Seasons is every season, by number.
func (q *PostgresQuery) Seasons(ctx context.Context) (*gamemasterv1.GetSeasonsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `SELECT number, starts_at, ends_at FROM seasons ORDER BY number`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &gamemasterv1.GetSeasonsResponse{}
	for rows.Next() {
		var (
			number           int64
			startsAt, endsAt time.Time
		)
		if err := rows.Scan(&number, &startsAt, &endsAt); err != nil {
			return nil, fmt.Errorf("failed to read a season: %w", err)
		}
		answer.Seasons = append(answer.Seasons, &gamemasterv1.Season{
			Number: uint32(number), StartsAt: timestamppb.New(startsAt), EndsAt: timestamppb.New(endsAt), //nolint:gosec // CHECK (number >= 0 AND number <= 4294967295).
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the seasons: %w", err)
	}
	return answer, nil
}
