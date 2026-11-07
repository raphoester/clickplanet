package raid_mode_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// The rule is raids.RaidMode.RunsAt's: started, and not ended yet.
const running = `SELECT started_at, ends_at, reason FROM raid_mode WHERE started_at <= $1 AND ends_at > $1`

func (q *PostgresQuery) RaidMode(ctx context.Context) (*chatv1.GetRaidModeResponse, error) {
	var (
		startedAt, endsAt time.Time
		reason            string
	)
	err := q.db.On(ctx).QueryRowContext(ctx, running, q.clock.Now().UTC()).Scan(&startedAt, &endsAt, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return &chatv1.GetRaidModeResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the raid mode: %w", err)
	}
	return &chatv1.GetRaidModeResponse{RaidMode: &chatv1.RaidMode{
		StartedAt: timestamppb.New(startedAt), EndsAt: timestamppb.New(endsAt), Reason: reason,
	}}, nil
}
