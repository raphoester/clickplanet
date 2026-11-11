package click_tried_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/click_tried_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type presence struct {
	clicks []bonuses.Click
}

func (p *presence) Clicked(click bonuses.Click) {
	p.clicks = append(p.clicks, click)
}

var (
	born  = time.Date(2026, 11, 11, 20, 0, 0, 0, time.UTC)
	tried = born.Add(3 * time.Minute)
)

func heard(t *testing.T, tried *planetv1.ClickTried) []bonuses.Click {
	t.Helper()

	p := &presence{}
	require.NoError(t, click_tried_subscriber.New(p).Handle(t.Context(), tried))
	return p.clicks
}

// What bonus_click heard inside planet's chain, before the offers left it: a take, a strike, an unchanged tile.
func TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard(t *testing.T) {
	for outcome, name := range planetv1.ClickOutcome_name {
		clicks := heard(t, &planetv1.ClickTried{Scope: "1.2.3.4", Outcome: planetv1.ClickOutcome(outcome)})

		switch planetv1.ClickOutcome(outcome) {
		case planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN, planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK,
			planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED:
			assert.Len(t, clicks, 1, name)
		default:
			assert.Empty(t, clicks, name)
		}
	}
}

func TestAClickSaysItsCallerWhenItWasTriedTheAccountsAgeAndWhetherItRanDry(t *testing.T) {
	account := cpsession.AccountCreatedAt(born).String()

	clicks := heard(t, &planetv1.ClickTried{
		Scope: "1.2.3.4", AccountId: account, Linked: true, TriedAt: timestamppb.New(tried),
		Outcome: planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN, Dry: true,
	})

	require.Len(t, clicks, 1)
	assert.Equal(t, bonuses.Click{
		Caller: bonuses.Caller{Scope: "1.2.3.4", Account: bonuses.Account(account), Linked: true},
		At:     tried,
		Born:   born,
		Dry:    true,
	}, clicks[0])
}

func TestAnAccountWhoseIdSaysNothingHasNoAge(t *testing.T) {
	clicks := heard(t, &planetv1.ClickTried{
		Scope: "1.2.3.4", AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Outcome: planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN,
	})

	require.Len(t, clicks, 1)
	assert.True(t, clicks[0].Born.IsZero())
}
