package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ordinaryWindow = 5 * time.Minute
	newFor         = 10 * time.Minute
	firstBoxAfter  = time.Minute
)

var (
	newcomerPayer = payer{Scope: "scope-n", Account: "newcomer", Created: epoch}
	veteranPayer  = payer{Scope: "scope-v", Account: "veteran", Created: epoch.Add(-24 * time.Hour)}
	everyWeight   = map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3}
)

func newWelcomingRegistry(t *testing.T, kinds map[Kind]float64, welcome WelcomeRules) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newRuledRegistry(func(rules *Rules) {
		rules.Kinds = kinds
		rules.Welcome = welcome
		rules.Boxes = Pace{MinInterval: ordinaryWindow, MaxInterval: ordinaryWindow, MaxChargesPerHour: 6}
	})
}

func welcoming(t *testing.T, kinds map[Kind]float64) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newWelcomingRegistry(t, kinds, WelcomeRules{Within: newFor, BoxAfter: firstBoxAfter})
}

type received struct {
	boxes []*Offer
}

func receivedOn(events <-chan Event) received {
	var got received
	for {
		select {
		case event := <-events:
			if event.Offer != nil {
				got.boxes = append(got.boxes, event.Offer)
			}
		default:
			return got
		}
	}
}

func after(t *testing.T, r *Registry, clock *cptime.FixedClock, d time.Duration) {
	t.Helper()

	clock.Advance(d)
	r.sweep(t.Context())
}

func TestANewcomersFirstBoxIsASpreadAMinuteAfterItsAccountWasMade(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter-time.Second)
	require.Empty(t, receivedOn(events).boxes, "not before the minute")

	after(t, registry, clock, time.Second)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	assert.Equal(t, KindSpreadClicks, boxes[0].Kind)
}

func TestTheFirstMinuteCountsFromTheAccountNotFromItsLaterClick(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, entrantOf(newcomerPayer))

	clock.Advance(3 * time.Minute)
	registry.Clicked(clickOf(newcomerPayer))
	registry.sweep(t.Context())

	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1, "a minute after it was made has passed already")
	assert.Equal(t, KindSpreadClicks, boxes[0].Kind)
}

func TestANewcomerIsWelcomedWithOneBoxOnly(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	entrant := entrantOf(newcomerPayer)
	events := attend(t, registry, entrant)
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	_, claimed := registry.Claim(boxes[0].Token, entrant, newcomerPayer.Scope)
	require.True(t, claimed)

	registry.Clicked(clickOf(newcomerPayer))
	after(t, registry, clock, ordinaryWindow-time.Second)
	assert.Empty(t, receivedOn(events).boxes, "the next box waits the ordinary window after the claim")

	registry.Clicked(clickOf(newcomerPayer))
	after(t, registry, clock, time.Second)
	assert.Len(t, receivedOn(events).boxes, 1)
}

func TestAnOlderAccountWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, entrantOf(veteranPayer))
	registry.Clicked(clickOf(veteranPayer))

	after(t, registry, clock, firstBoxAfter)
	assert.Empty(t, receivedOn(events).boxes)

	registry.Clicked(clickOf(veteranPayer))
	after(t, registry, clock, ordinaryWindow-firstBoxAfter)
	assert.Len(t, receivedOn(events).boxes, 1)
}

func TestAnAccountThatSaysNothingOfItsAgeIsNotWelcomed(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	unknown := payer{Scope: "scope-u", Account: "unknown"}
	events := attend(t, registry, entrantOf(unknown))
	registry.Clicked(clickOf(unknown))

	after(t, registry, clock, firstBoxAfter)

	assert.Empty(t, receivedOn(events).boxes)
}

func TestWithNoWelcomeANewcomerWaitsLikeAnybody(t *testing.T) {
	registry, clock := newWelcomingRegistry(t, everyWeight, WelcomeRules{})
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter)

	assert.Empty(t, receivedOn(events).boxes)
}

func TestAWelcomeKindTheConfigLeavesOutGivesNoEarlyBoxAndNoEarlyRefill(t *testing.T) {
	registry, clock := welcoming(t, map[Kind]float64{KindRefill: 1})
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter)

	assert.Empty(t, receivedOn(events).boxes, "a fresh account is never handed a bank early")

	registry.Clicked(clickOf(newcomerPayer))
	after(t, registry, clock, ordinaryWindow)
	assert.Len(t, receivedOn(events).boxes, 1, "the ordinary schedule goes on")
}

func TestANewcomerIsForgottenOnceItIsNoLongerNew(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))
	require.Len(t, registry.newcomers, 1)

	after(t, registry, clock, newFor)

	assert.Empty(t, registry.newcomers)
}
