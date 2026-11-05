package teaser_query

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/promoread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var ErrNoBattle = errors.New("no battle has this id")

// Battle is what a battle plays: its rule set, and its points' times.
type Battle struct {
	Set   string
	Times uint32
}

// Battles is what a battle plays, by the id the gamemaster's timeline names it by: ErrNoBattle for none.
type Battles interface {
	Battle(ctx context.Context, battleID string) (Battle, error)
}

// Points is a battle's points by its times: placeholders/rpc_scoring_points.
type Points interface {
	Battle(ctx context.Context, times uint32) (placeholders.Battle, error)
}

// Planet is the values a set plays: cards/rpc_planet_rules, cards.ErrUnknownSet for a set it does not keep.
type Planet interface {
	Played(ctx context.Context, set string) (*planetv1.Rules, error)
}

type Links interface {
	PublicURL(id backgrounds.BackgroundID) string
}

func NewPostgresQuery(db cppg.Conn, battles Battles, points Points, planet Planet, links Links, logger *slog.Logger) *PostgresQuery {
	return &PostgresQuery{db: db, battles: battles, points: points, planet: planet, links: links, logger: logger}
}

type PostgresQuery struct {
	db      cppg.Conn
	battles Battles
	points  Points
	planet  Planet
	links   Links
	logger  *slog.Logger
}

type written struct {
	background backgrounds.BackgroundID
	headline   string
	line       string
}

// Teaser is a battle's cards, filled with what it and its set play now. A card that does not fill is left out, and so
// is every card of a set planet no longer keeps: each is logged at Error, and the other cards still show.
func (q *PostgresQuery) Teaser(ctx context.Context, battleID string) (*promov1.GetTeaserResponse, error) {
	battle, err := q.battles.Battle(ctx, battleID)
	if err != nil {
		return nil, fmt.Errorf("failed to read what the battle %q plays: %w", battleID, err)
	}
	set := battle.Set
	drafts, err := q.drafts(ctx, set)
	if err != nil {
		return nil, err
	}
	out := &promov1.GetTeaserResponse{}
	if len(drafts) == 0 {
		return out, nil
	}

	played, err := q.planet.Played(ctx, set)
	switch {
	case errors.Is(err, cards.ErrUnknownSet):
		q.logger.ErrorContext(ctx, "a teaser's set is no longer kept by planet", slog.String("set", set),
			slog.String("battle", battleID), slog.Any("error", err))
		return out, nil
	case err != nil:
		return nil, fmt.Errorf("failed to read what %q plays: %w", set, err)
	}
	points, err := q.points.Battle(ctx, battle.Times)
	if err != nil {
		return nil, fmt.Errorf("failed to read the points of the battle %q: %w", battleID, err)
	}

	for i, draft := range drafts {
		filled, err := promoread.Filled(q.links.PublicURL(draft.background), draft.headline, draft.line, played, points)
		if err != nil {
			q.logger.ErrorContext(ctx, "a teaser card does not fill", slog.String("set", set), slog.Int("card", i+1),
				slog.String("battle", battleID), slog.Any("error", err))
			continue
		}
		out.Cards = append(out.Cards, filled)
	}
	return out, nil
}

func (q *PostgresQuery) drafts(ctx context.Context, set string) ([]written, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT background_id, headline, line FROM cards WHERE rule_set = $1 ORDER BY position`, set)
	if err != nil {
		return nil, fmt.Errorf("failed to read the cards of %q: %w", set, err)
	}
	defer func() { _ = rows.Close() }()

	var drafts []written
	for rows.Next() {
		var (
			id    uuid.UUID
			draft written
		)
		if err := rows.Scan(&id, &draft.headline, &draft.line); err != nil {
			return nil, fmt.Errorf("failed to read a card of %q: %w", set, err)
		}
		draft.background = backgrounds.BackgroundID(id)
		drafts = append(drafts, draft)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the cards of %q: %w", set, err)
	}
	return drafts, nil
}
