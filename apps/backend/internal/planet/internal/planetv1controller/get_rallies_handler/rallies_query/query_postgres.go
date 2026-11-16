package rallies_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock}
}

type PostgresQuery struct {
	db    cppg.Conn
	clock cptime.Clock
}

// Rallies is every rally still on, each with the time it has left now, the soonest to end first.
func (q *PostgresQuery) Rallies(ctx context.Context) (*planetv1.GetRalliesResponse, error) {
	now := q.clock.Now()
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT account, id, flag, x, y, z, radius, called_at, ends_at FROM rallies WHERE ends_at > $1 ORDER BY ends_at, id
	`, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to read the rallies on: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetRalliesResponse{}
	for rows.Next() {
		var (
			account, id uuid.UUID
			rally       rallies.Snapshot
		)
		if err := rows.Scan(&account, &id, &rally.Flag, &rally.Centre.X, &rally.Centre.Y, &rally.Centre.Z, &rally.Radius,
			&rally.CalledAt, &rally.EndsAt); err != nil {
			return nil, fmt.Errorf("failed to read a rally: %w", err)
		}
		rally.Account, rally.ID = cpsession.AccountID(account), rallies.RallyID(id)
		answer.Rallies = append(answer.Rallies, planetmessage.Rally(rally, rally.EndsAt.Sub(now)))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the rallies on: %w", err)
	}

	return answer, nil
}
