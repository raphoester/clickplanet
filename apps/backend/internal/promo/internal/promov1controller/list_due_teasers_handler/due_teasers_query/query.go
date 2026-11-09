package due_teasers_query

import (
	"context"
	"fmt"
	"slices"
	"time"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// ListTeasers' own query: teasers_query.
type Teasers interface {
	Teasers(ctx context.Context, within time.Duration) ([]*promov1.Teaser, error)
}

// The publishing module's posts: rpc_publishing.
type Posts interface {
	Teased(ctx context.Context, since time.Time) ([]teasers.BattleID, error)
}

func New(listed Teasers, posts Posts, clock cptime.Clock) Query {
	return Query{listed: listed, posts: posts, clock: clock}
}

// The battles whose teaser is due, as ListTeasers lists them.
type Query struct {
	listed Teasers
	posts  Posts
	clock  cptime.Clock
}

// Those within teasers.Ahead that no post since teasers.TeasedSince names, by start.
func (q Query) Due(ctx context.Context) ([]*promov1.Teaser, error) {
	listed, err := q.listed.Teasers(ctx, teasers.Ahead)
	if err != nil {
		return nil, fmt.Errorf("failed to list the battles to come: %w", err)
	}
	teased, err := q.posts.Teased(ctx, teasers.TeasedSince(q.clock.Now()))
	if err != nil {
		return nil, fmt.Errorf("failed to read the battles teased already: %w", err)
	}
	due := make([]*promov1.Teaser, 0, len(listed))
	for _, teaser := range listed {
		if !slices.Contains(teased, teasers.BattleID(teaser.GetBattleId())) {
			due = append(due, teaser)
		}
	}
	return due, nil
}
