// Package cards is what each rule set shows in a battle's teaser: up to five cards, each a background of the library
// and words over it, written with placeholders the set's values fill when a player reads them.
package cards

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

// SetName is a rule set's name, as planet and the antibot keep it.
type SetName string

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

const (
	MaxCards    = 5
	maxHeadline = 40
	maxLine     = 120
)

var (
	ErrInvalidCards = errors.New("not cards a set may show")
	ErrUnknownSet   = errors.New("planet keeps no rule set of this name")
)

// Draft is a card as the operator writes it.
type Draft struct {
	Background backgrounds.BackgroundID
	Headline   string
	Line       string
}

type Card struct {
	background backgrounds.BackgroundID
	headline   string
	line       string
}

// Deck is a set's cards, in order. None is a set whose battles are not teased.
type Deck struct {
	set   SetName
	cards []Card
}

type CardSnapshot struct {
	Background backgrounds.BackgroundID
	Headline   string
	Line       string
}

type DeckSnapshot struct {
	Set   SetName
	Cards []CardSnapshot
}

// Every reason at once. Words are measured as written, placeholders and all.
func NewDeck(set SetName, drafts []Draft) (Deck, error) {
	var reasons []string
	if !namePattern.MatchString(string(set)) {
		reasons = append(reasons, fmt.Sprintf("%q is no rule set's name", set))
	}
	if len(drafts) > MaxCards {
		reasons = append(reasons, fmt.Sprintf("%d cards, and a set shows %d at most", len(drafts), MaxCards))
	}
	deck := Deck{set: set, cards: make([]Card, 0, len(drafts))}
	for i, draft := range drafts {
		card := Card{background: draft.Background, headline: strings.TrimSpace(draft.Headline), line: strings.TrimSpace(draft.Line)}
		for _, reason := range card.reasons() {
			reasons = append(reasons, fmt.Sprintf("card %d: %s", i+1, reason))
		}
		deck.cards = append(deck.cards, card)
	}
	if len(reasons) > 0 {
		return Deck{}, fmt.Errorf("%w: %s", ErrInvalidCards, strings.Join(reasons, "; "))
	}
	return deck, nil
}

func (c Card) reasons() []string {
	var reasons []string
	if c.background == (backgrounds.BackgroundID{}) {
		reasons = append(reasons, "no background")
	}
	if c.headline == "" || utf8.RuneCountInString(c.headline) > maxHeadline {
		reasons = append(reasons, fmt.Sprintf("a headline of 1 to %d characters", maxHeadline))
	}
	if utf8.RuneCountInString(c.line) > maxLine {
		reasons = append(reasons, fmt.Sprintf("a line of %d characters at most", maxLine))
	}
	for _, words := range []string{c.headline, c.line} {
		if err := placeholders.Check(words); err != nil {
			reasons = append(reasons, err.Error())
		}
	}
	return reasons
}

func (d Deck) TakeSnapshot() DeckSnapshot {
	snapshot := DeckSnapshot{Set: d.set, Cards: make([]CardSnapshot, 0, len(d.cards))}
	for _, card := range d.cards {
		snapshot.Cards = append(snapshot.Cards, CardSnapshot{Background: card.background, Headline: card.headline, Line: card.line})
	}
	return snapshot
}
