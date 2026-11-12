package save_cards_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards/usecases/save_cards_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	planet = cards.FakeSets{"default", "urf"}
	war    = backgrounds.BackgroundID{15: 1}
	drafts = []cards.Draft{{Background: war, Headline: "Clicks refill {clicks.tempo}× faster"}}
)

func TestASetsCardsAreKept(t *testing.T) {
	store := cards.NewMemoryStore()

	require.NoError(t, save_cards_usecase.New(planet, store, cpatomic.FakeTransactioner{}).Execute(t.Context(), "urf", drafts))

	deck, ok := store.Deck("urf")
	require.True(t, ok)
	assert.Equal(t, cards.DeckSnapshot{Set: "urf", Cards: []cards.CardSnapshot{
		{Background: war, Headline: "Clicks refill {clicks.tempo}× faster"},
	}}, deck)
}

func TestASetPlanetDoesNotKeepShowsNothing(t *testing.T) {
	store := cards.NewMemoryStore()

	err := save_cards_usecase.New(planet, store, cpatomic.FakeTransactioner{}).Execute(t.Context(), "rush", drafts)

	require.ErrorIs(t, err, cards.ErrUnknownSet)
	_, ok := store.Deck("rush")
	assert.False(t, ok)
}

func TestCardsTheRulesRefuseAreNotKept(t *testing.T) {
	store := cards.NewMemoryStore()

	err := save_cards_usecase.New(planet, store, cpatomic.FakeTransactioner{}).Execute(t.Context(), "urf",
		[]cards.Draft{{Background: war}})

	require.ErrorIs(t, err, cards.ErrInvalidCards)
	_, ok := store.Deck("urf")
	assert.False(t, ok)
}
