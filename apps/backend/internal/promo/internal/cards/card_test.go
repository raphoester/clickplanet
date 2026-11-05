package cards_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

var war = backgrounds.BackgroundID{15: 1}

func draft(headline, line string) cards.Draft {
	return cards.Draft{Background: war, Headline: headline, Line: line}
}

func TestADeckKeepsItsCardsInOrder(t *testing.T) {
	deck, err := cards.NewDeck("urf", []cards.Draft{
		draft("  Clicks refill {clicks.tempo}× faster ", " Every country at once "),
		draft("A box every {bonuses.boxes.minInterval}", ""),
	})

	require.NoError(t, err)
	assert.Equal(t, cards.DeckSnapshot{Set: "urf", Cards: []cards.CardSnapshot{
		{Background: war, Headline: "Clicks refill {clicks.tempo}× faster", Line: "Every country at once"},
		{Background: war, Headline: "A box every {bonuses.boxes.minInterval}"},
	}}, deck.TakeSnapshot())
}

func TestNoCardsIsASetNotTeased(t *testing.T) {
	deck, err := cards.NewDeck("urf", nil)

	require.NoError(t, err)
	assert.Empty(t, deck.TakeSnapshot().Cards)
}

func TestEveryReasonIsGivenAtOnce(t *testing.T) {
	_, err := cards.NewDeck("URF!", []cards.Draft{
		draft("", strings.Repeat("a", 121)),
		{Headline: strings.Repeat("a", 41)},
		draft("{clicks.speed}× faster", "an { alone"),
		draft("ok", ""), draft("ok", ""), draft("ok", ""),
	})

	require.ErrorIs(t, err, cards.ErrInvalidCards)
	for _, reason := range []string{
		`"URF!" is no rule set's name`,
		"6 cards, and a set shows 5 at most",
		"card 1: a headline of 1 to 40 characters",
		"card 1: a line of 120 characters at most",
		"card 2: no background",
		"card 2: a headline of 1 to 40 characters",
		"card 3: " + placeholders.ErrInvalidPlaceholder.Error() + ": {clicks.speed} names no number or duration of the rules",
		"card 3: " + placeholders.ErrInvalidPlaceholder.Error() + ": a brace opens or closes no placeholder",
	} {
		assert.Contains(t, err.Error(), reason)
	}
}

func TestWordsAreMeasuredAsWritten(t *testing.T) {
	_, err := cards.NewDeck("urf", []cards.Draft{draft(strings.Repeat("é", 40), strings.Repeat("ü", 120))})

	require.NoError(t, err, "characters, not bytes")
}
