package bombs_and_fortifies_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Landmasses interface {
	TilesOf(landmass clicks.LandmassID) []uint32
}

func NewPostgresQuery(db cppg.Querier, landmasses Landmasses) *PostgresQuery {
	return &PostgresQuery{db: db, landmasses: landmasses}
}

type PostgresQuery struct {
	db         cppg.Querier
	landmasses Landmasses
}

const kept = `
	SELECT account, kind = 'bomb', COALESCE((payload->>'landmass')::integer, 0), count(*)
	FROM ledger_events
	WHERE account = ANY($1::uuid[]) AND kind IN ('bomb', 'fortify')
	GROUP BY 1, 2, 3
	ORDER BY 1
`

func (q *PostgresQuery) BombsAndFortifies(ctx context.Context, asked []cpsession.AccountID) (*planetv1.GetBombsAndFortifiesResponse, error) {
	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}

	rows, err := q.db.QueryContext(ctx, kept, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to read the accounts' bombs and fortifies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &planetv1.GetBombsAndFortifiesResponse{}
	var last *planetv1.BombsAndFortifies
	for rows.Next() {
		var (
			account  uuid.UUID
			bomb     bool
			landmass uint16
			events   uint64
		)
		if err := rows.Scan(&account, &bomb, &landmass, &events); err != nil {
			return nil, fmt.Errorf("failed to read an account's bombs or fortifies: %w", err)
		}
		if last == nil || last.GetAccountId() != account.String() {
			last = &planetv1.BombsAndFortifies{AccountId: account.String()}
			answer.Accounts = append(answer.Accounts, last)
		}
		if bomb {
			last.BombsDropped += events
		} else {
			last.TilesFortified += events * uint64(len(q.landmasses.TilesOf(clicks.LandmassID(landmass))))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the accounts' bombs and fortifies: %w", err)
	}
	return answer, nil
}
