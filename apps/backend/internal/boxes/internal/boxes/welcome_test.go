package boxes

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
)

func newWelcomingRegistry(t *testing.T, welcome WelcomeRules) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newRuledRegistry(func(rules *Rules) {
		rules.Welcome = welcome
		rules.Pace = Pace{MinInterval: ordinaryWindow, MaxInterval: ordinaryWindow, MaxBoxesPerHour: 6}
	})
}

func welcoming(t *testing.T) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newWelcomingRegistry(t, WelcomeRules{Within: newFor, BoxAfter: firstBoxAfter})
}

type received struct {
	boxes []Offer
}

func receivedOn(offers <-chan Offer) received {
	var got received
	for {
		select {
		case offer := <-offers:
			got.boxes = append(got.boxes, offer)
		default:
			return got
		}
	}
}

// welcomed claims the box as the payer, and says whether it was its welcome.
func welcomed(t *testing.T, r *Registry, p payer, offer Offer) bool {
	t.Helper()

	caught, claimed := r.Claim(offer.Token, entrantOf(p), p.Scope)
	require.True(t, claimed)
	return caught.Welcome
}

func after(t *testing.T, r *Registry, clock *cptime.FixedClock, d time.Duration) {
	t.Helper()

	clock.Advance(d)
	r.sweep()
}

func TestANewcomersFirstBoxIsAWelcomeAMinuteAfterItsAccountWasMade(t *testing.T) {
	registry, clock := welcoming(t)
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter-time.Second)
	require.Empty(t, receivedOn(events).boxes, "not before the minute")

	after(t, registry, clock, time.Second)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	assert.True(t, welcomed(t, registry, newcomerPayer, boxes[0]), "the bonuses module pays it a spread")
}

func TestTheFirstMinuteCountsFromTheAccountNotFromItsLaterClick(t *testing.T) {
	registry, clock := welcoming(t)
	events := attend(t, registry, entrantOf(newcomerPayer))

	clock.Advance(3 * time.Minute)
	registry.Clicked(clickOf(newcomerPayer))
	registry.sweep()

	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1, "a minute after it was made has passed already")
	assert.True(t, welcomed(t, registry, newcomerPayer, boxes[0]))
}

func TestANewcomerIsWelcomedWithOneBoxOnly(t *testing.T) {
	registry, clock := welcoming(t)
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	require.True(t, welcomed(t, registry, newcomerPayer, boxes[0]))

	registry.Clicked(clickOf(newcomerPayer))
	after(t, registry, clock, ordinaryWindow-time.Second)
	assert.Empty(t, receivedOn(events).boxes, "the next box waits the ordinary window after the claim")

	registry.Clicked(clickOf(newcomerPayer))
	after(t, registry, clock, time.Second)
	boxes = receivedOn(events).boxes
	require.Len(t, boxes, 1)
	assert.False(t, welcomed(t, registry, newcomerPayer, boxes[0]), "the second box is an ordinary one")
}

func TestAnOlderAccountWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := welcoming(t)
	events := attend(t, registry, entrantOf(veteranPayer))
	registry.Clicked(clickOf(veteranPayer))

	after(t, registry, clock, firstBoxAfter)
	assert.Empty(t, receivedOn(events).boxes)

	registry.Clicked(clickOf(veteranPayer))
	after(t, registry, clock, ordinaryWindow-firstBoxAfter)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	assert.False(t, welcomed(t, registry, veteranPayer, boxes[0]))
}

func TestAnAccountThatSaysNothingOfItsAgeIsNotWelcomed(t *testing.T) {
	registry, clock := welcoming(t)
	unknown := payer{Scope: "scope-u", Account: "unknown"}
	events := attend(t, registry, entrantOf(unknown))
	registry.Clicked(clickOf(unknown))

	after(t, registry, clock, firstBoxAfter)

	assert.Empty(t, receivedOn(events).boxes)
}

func TestWithNoWelcomeANewcomerWaitsLikeAnybody(t *testing.T) {
	registry, clock := newWelcomingRegistry(t, WelcomeRules{})
	events := attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))

	after(t, registry, clock, firstBoxAfter)

	assert.Empty(t, receivedOn(events).boxes)
}

func TestANewcomerIsForgottenOnceItIsNoLongerNew(t *testing.T) {
	registry, clock := welcoming(t)
	attend(t, registry, entrantOf(newcomerPayer))
	registry.Clicked(clickOf(newcomerPayer))
	require.Len(t, registry.newcomers, 1)

	after(t, registry, clock, newFor)

	assert.Empty(t, registry.newcomers)
}
