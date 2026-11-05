package cards_query_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/postgres_background_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/postgres_card_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_cards_handler/cards_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var war = backgrounds.BackgroundID{15: 1}

type planet struct {
	names  []string
	played map[string]*planetv1.Rules
}

func (p planet) Names(context.Context) ([]string, error) { return p.names, nil }

func (p planet) Played(_ context.Context, set string) (*planetv1.Rules, error) {
	played, ok := p.played[set]
	if !ok {
		return nil, fmt.Errorf("%w: %q", cards.ErrUnknownSet, set)
	}
	return played, nil
}

func TestEverySetPlanetKeepsIsListedWithItsCardsAsWrittenAndAsRead(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "promo", migrations.FS)
	background, err := backgrounds.Keep(backgrounds.File{ID: war, MediaType: "image/png", Size: 1}, "The war", time.Now(),
		backgrounds.Rules{MaxBytes: 2 << 20})
	require.NoError(t, err)
	require.NoError(t, postgres_background_store.New(db).Keep(t.Context(), background))
	deck, err := cards.NewDeck("urf", []cards.Draft{
		{Background: war, Headline: "Clicks refill {clicks.tempo}× faster", Line: "Every country at once."},
		{Background: war, Headline: "Bombs blast {bonuses.bomb.rings} rings"},
	})
	require.NoError(t, err)
	require.NoError(t, postgres_card_store.New(db).Save(t.Context(), deck))
	defaults := &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(1)}}
	urf := &planetv1.Rules{Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3)}}

	answer, err := cards_query.NewPostgresQuery(db,
		planet{names: []string{"default", "urf"}, played: map[string]*planetv1.Rules{"default": defaults, "urf": urf}},
		backgrounds.NewMemoryBucket(),
	).Cards(t.Context())

	require.NoError(t, err)
	require.Len(t, answer.GetSets(), 2)
	assert.True(t, proto.Equal(&promov1.SetCards{RuleSet: "default"}, answer.GetSets()[0]))
	urfCards := answer.GetSets()[1].GetCards()
	assert.Equal(t, "urf", answer.GetSets()[1].GetRuleSet())
	require.Len(t, urfCards, 2)
	assert.True(t, proto.Equal(&promov1.Card{
		Draft: &promov1.CardDraft{BackgroundId: war.String(), Headline: "Clicks refill {clicks.tempo}× faster", Line: "Every country at once."},
		Filled: &promov1.TeaserCard{
			BackgroundUrl: "https://media.example/promo/backgrounds/" + war.String(), Headline: "Clicks refill 3× faster",
			Line: "Every country at once.",
		},
	}, urfCards[0]), "%v", urfCards[0])
	assert.Nil(t, urfCards[1].GetFilled(), "the set plays no rings")
	assert.Contains(t, urfCards[1].GetProblem(), "{bonuses.bomb.rings}")

	assert.Len(t, answer.GetPlaceholders(), len(placeholders.Paths()))
	assert.True(t, proto.Equal(&promov1.Placeholder{Path: "clicks.tempo", Example: "1"}, answer.GetPlaceholders()[0]))
}

func TestASetWhoseValuesPlanetWillNotTellSaysWhy(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "promo", migrations.FS)
	background, err := backgrounds.Keep(backgrounds.File{ID: war, MediaType: "image/png", Size: 1}, "The war", time.Now(),
		backgrounds.Rules{MaxBytes: 2 << 20})
	require.NoError(t, err)
	require.NoError(t, postgres_background_store.New(db).Keep(t.Context(), background))
	deck, err := cards.NewDeck("urf", []cards.Draft{{Background: war, Headline: "War"}})
	require.NoError(t, err)
	require.NoError(t, postgres_card_store.New(db).Save(t.Context(), deck))

	answer, err := cards_query.NewPostgresQuery(db,
		planet{names: []string{"default", "urf"}, played: map[string]*planetv1.Rules{"default": {}}},
		backgrounds.NewMemoryBucket(),
	).Cards(t.Context())

	require.NoError(t, err)
	card := answer.GetSets()[1].GetCards()[0]
	assert.Nil(t, card.GetFilled())
	assert.Contains(t, card.GetProblem(), cards.ErrUnknownSet.Error())
}
