package renewals_query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/shopv1controller/renewalmessage"
)

func NewPostgresQuery(db cppg.Conn) PostgresQuery {
	return PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// Mine is the buyer's renewals that renew or stop at their period's end, the soonest first.
func (q PostgresQuery) Mine(ctx context.Context, buyer orders.AccountID) (*shopv1.ListMyRenewalsResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT id, plan::text, target_kind::text, target, state::text, period_ends_at FROM renewals
		WHERE account_id = $1 AND state <> 'ended' ORDER BY period_ends_at, id
	`, uuid.UUID(buyer))
	if err != nil {
		return nil, fmt.Errorf("failed to read the renewals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	mine := &shopv1.ListMyRenewalsResponse{}
	for rows.Next() {
		var id, plan, kind, target, state string
		var ends time.Time
		if err := rows.Scan(&id, &plan, &kind, &target, &state, &ends); err != nil {
			return nil, fmt.Errorf("failed to read a renewal: %w", err)
		}
		kept, err := orders.TargetOf(orders.TargetKind(kind), target)
		if err != nil {
			return nil, fmt.Errorf("the renewal %s: %w", id, err)
		}
		mine.Renewals = append(mine.Renewals, renewalmessage.Renewal(renewals.Snapshot{
			ID: renewals.SubscriptionID(id), Order: orders.Snapshot{Buyer: buyer, Plan: plans.Plan(plan), Target: kept},
			State: renewals.State(state), PeriodEndsAt: ends.UTC(),
		}))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the renewals: %w", err)
	}
	return mine, nil
}
