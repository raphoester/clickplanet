package teasers_query

import (
	"context"
	"errors"
	"fmt"
	"time"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The battles to come, in the gamemaster's words: rpc_gamemaster_timeline.
type Timeline interface {
	Battles(ctx context.Context) ([]*gamemasterv1.Battle, error)
}

// A battle's cards as players read them: the teaser's own query, teaser_query.
type Cards interface {
	Teaser(ctx context.Context, battleID string) (*promov1.GetTeaserResponse, error)
}

func New(timeline Timeline, cards Cards, clock cptime.Clock) Query {
	return Query{timeline: timeline, cards: cards, clock: clock}
}

// What the game would tease of each battle to come, as GetTeaser fills it, with the battle's facts beside.
type Query struct {
	timeline Timeline
	cards    Cards
	clock    cptime.Clock
}

// The battles that start within the window from now, by start, those whose set has cards alone. A battle the schedule
// dropped since the timeline was read is left out.
func (q Query) Teasers(ctx context.Context, within time.Duration) ([]*promov1.Teaser, error) {
	battles, err := q.timeline.Battles(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the adapter already says what it asked.
	}

	end := q.clock.Now().Add(within)
	var teasers []*promov1.Teaser
	for _, battle := range battles {
		if battle.GetStartsAt().AsTime().After(end) {
			continue
		}
		id := battle.GetEntryId()
		if id == "" {
			id = battle.GetRoundId()
		}
		teaser, err := q.cards.Teaser(ctx, id)
		switch {
		case errors.Is(err, teaser_query.ErrNoBattle):
			continue
		case err != nil:
			return nil, fmt.Errorf("failed to read the cards of the battle %q: %w", id, err)
		case len(teaser.GetCards()) == 0:
			continue
		}
		teasers = append(teasers, &promov1.Teaser{
			BattleId: id, StartsAt: battle.GetStartsAt(), EndsAt: battle.GetEndsAt(), Description: battle.GetDescription(),
			Times: battle.GetTimes(), EndsSeason: battle.GetEndsSeason(), Cards: teaser.GetCards(),
		})
	}
	return teasers, nil
}
