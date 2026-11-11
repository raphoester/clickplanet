package cards_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/promoread"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

// Planet is the sets planet keeps and the values each plays: cards/rpc_rule_sets.
type Planet interface {
	Names(ctx context.Context) ([]string, error)
	Played(ctx context.Context, set string) (placeholders.Played, error)
}

// Points is a battle's points by its times: placeholders/rpc_scoring_points.
type Points interface {
	Battle(ctx context.Context, times uint32) (placeholders.Battle, error)
}

type Links interface {
	PublicURL(id backgrounds.BackgroundID) string
}

const (
	// The set whose values a placeholder's example reads.
	defaultSet = "default"
	// The battle a set's cards are filled as: the Final Battle's times.
	exampleTimes = 3
)

func NewPostgresQuery(db cppg.Conn, planet Planet, points Points, links Links) *PostgresQuery {
	return &PostgresQuery{db: db, planet: planet, points: points, links: links}
}

type PostgresQuery struct {
	db     cppg.Conn
	planet Planet
	points Points
	links  Links
}

// Cards is every set planet keeps with its cards, as written and as players read them now in the example battle. A set
// whose values planet will not tell, or a card that does not fill, says why instead.
func (q *PostgresQuery) Cards(ctx context.Context) (*promov1.GetCardsResponse, error) {
	names, err := q.planet.Names(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rule sets: %w", err)
	}
	drafts, err := q.drafts(ctx)
	if err != nil {
		return nil, err
	}
	defaults, err := q.planet.Played(ctx, defaultSet)
	if err != nil {
		return nil, fmt.Errorf("failed to read what default plays: %w", err)
	}
	battle, err := q.points.Battle(ctx, exampleTimes)
	if err != nil {
		return nil, fmt.Errorf("failed to read the points of a battle of %d times: %w", exampleTimes, err)
	}

	out := &promov1.GetCardsResponse{
		Placeholders:  examples(placeholders.RulePaths(), defaults, battle),
		ExampleBattle: &promov1.ExampleBattle{Times: exampleTimes, Placeholders: examples(placeholders.BattlePaths(), defaults, battle)},
	}
	for _, name := range names {
		set := &promov1.SetCards{RuleSet: name}
		written := drafts[name]
		if len(written) > 0 {
			played, playedErr := q.planet.Played(ctx, name)
			for _, draft := range written {
				card := &promov1.Card{Draft: draft}
				switch {
				case playedErr != nil:
					card.Problem = playedErr.Error()
				default:
					filled, err := q.filled(draft, played, battle)
					if err != nil {
						card.Problem = err.Error()
					}
					card.Filled = filled
				}
				set.Cards = append(set.Cards, card)
			}
		}
		out.Sets = append(out.Sets, set)
	}
	return out, nil
}

func examples(paths []string, played placeholders.Played, battle placeholders.Battle) []*promov1.Placeholder {
	out := make([]*promov1.Placeholder, 0, len(paths))
	for _, path := range paths {
		example, err := placeholders.Fill("{"+path+"}", played, battle)
		if err != nil {
			example = ""
		}
		out = append(out, &promov1.Placeholder{Path: path, Example: example})
	}
	return out
}

func (q *PostgresQuery) filled(draft *promov1.CardDraft, played placeholders.Played, battle placeholders.Battle) (*promov1.TeaserCard, error) {
	id, err := uuid.Parse(draft.GetBackgroundId())
	if err != nil {
		return nil, fmt.Errorf("failed to read a background id: %w", err)
	}
	filled, err := promoread.Filled(q.links.PublicURL(backgrounds.BackgroundID(id)), draft.GetHeadline(), draft.GetLine(), played, battle)
	if err != nil {
		return nil, fmt.Errorf("failed to fill a card: %w", err)
	}
	return filled, nil
}

// Every set's cards as written, in order.
func (q *PostgresQuery) drafts(ctx context.Context) (map[string][]*promov1.CardDraft, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, `
		SELECT rule_set, background_id, headline, line FROM cards ORDER BY rule_set, position`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the cards: %w", err)
	}
	defer func() { _ = rows.Close() }()

	drafts := map[string][]*promov1.CardDraft{}
	for rows.Next() {
		var (
			set   string
			id    uuid.UUID
			draft promov1.CardDraft
		)
		if err := rows.Scan(&set, &id, &draft.Headline, &draft.Line); err != nil {
			return nil, fmt.Errorf("failed to read a card: %w", err)
		}
		draft.BackgroundId = id.String()
		drafts[set] = append(drafts[set], &draft)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the cards: %w", err)
	}
	return drafts, nil
}
