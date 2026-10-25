package charge_fills_query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const shown = 20

func NewPostgresQuery(db cppg.Querier) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Querier
}

func (q *PostgresQuery) ChargeFills(ctx context.Context) (*planetv1.GetChargeFillsResponse, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT request_id, source::text, started_at, accounts_filled, finished_at
		FROM charge_fills ORDER BY started_at DESC, request_id DESC LIMIT $1
	`, shown)
	if err != nil {
		return nil, fmt.Errorf("failed to read the charge fills: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetChargeFillsResponse{}
	for rows.Next() {
		var (
			request, source string
			startedAt       time.Time
			filled          uint32
			finishedAt      sql.NullTime
		)
		if err := rows.Scan(&request, &source, &startedAt, &filled, &finishedAt); err != nil {
			return nil, fmt.Errorf("failed to read a charge fill: %w", err)
		}

		encoded, err := sourceOf(source)
		if err != nil {
			return nil, err
		}
		fill := &planetv1.ChargeFill{
			RequestId:      request,
			Source:         encoded,
			StartedAt:      timestamppb.New(startedAt),
			AccountsFilled: filled,
		}
		if finishedAt.Valid {
			fill.FinishedAt = timestamppb.New(finishedAt.Time)
		}
		answer.Fills = append(answer.Fills, fill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the charge fills: %w", err)
	}

	return answer, nil
}

func sourceOf(source string) (planetv1.FillSource, error) {
	switch source {
	case "internal":
		return planetv1.FillSource_FILL_SOURCE_INTERNAL, nil
	case "operator":
		return planetv1.FillSource_FILL_SOURCE_OPERATOR, nil
	default:
		return planetv1.FillSource_FILL_SOURCE_UNSPECIFIED, fmt.Errorf("a charge fill came from %q", source)
	}
}
