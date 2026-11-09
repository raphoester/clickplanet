// The battles to come with their cards, as ListTeasers lists them, in the domain's words.
package listed_battles

import (
	"context"
	"time"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

// ListTeasers' own query: teasers_query.
type Query interface {
	Teasers(ctx context.Context, within time.Duration) ([]*promov1.Teaser, error)
}

func New(query Query) Battles {
	return Battles{query: query}
}

type Battles struct {
	query Query
}

func (b Battles) Teasers(ctx context.Context, within time.Duration) ([]teasers.Teaser, error) {
	listed, err := b.query.Teasers(ctx, within)
	if err != nil {
		return nil, err //nolint:wrapcheck // a pure delegation: the query already named what failed.
	}
	battles := make([]teasers.Teaser, 0, len(listed))
	for _, wire := range listed {
		battle := teasers.Teaser{
			Battle: teasers.BattleID(wire.GetBattleId()), StartsAt: wire.GetStartsAt().AsTime(), EndsAt: wire.GetEndsAt().AsTime(),
			Description: wire.GetDescription(), Times: wire.GetTimes(), EndsSeason: wire.GetEndsSeason(),
		}
		for _, card := range wire.GetCards() {
			battle.Cards = append(battle.Cards, teasers.Card{
				BackgroundURL: card.GetBackgroundUrl(), Headline: card.GetHeadline(), Line: card.GetLine(),
			})
		}
		battles = append(battles, battle)
	}
	return battles, nil
}
