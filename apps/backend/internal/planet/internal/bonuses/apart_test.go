package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func pacedRegistry(box, quiz time.Duration) (*Registry, *cptime.FixedClock) {
	return newRuledRegistry(func(rules *Rules) {
		rules.Boxes = Pace{MinInterval: box, MaxInterval: box, MaxChargesPerHour: 60}
		rules.Quiz = quizzing(quiz)
		rules.Quiz.MaxChargesPerHour = 60
	})
}

// until plays the caller clicking and the sweep running every second until the clock reads epoch+d.
func until(t *testing.T, r *Registry, clock *cptime.FixedClock, d time.Duration, entrant Entrant) {
	t.Helper()

	for clock.Now().Before(epoch.Add(d)) {
		clock.Advance(time.Second)
		clicked(r, entrant)
		r.sweep(t.Context())
	}
}

func TestTheSpacingIsHalfTheShorterShortestWait(t *testing.T) {
	registry, _ := pacedRegistry(4*time.Minute, 6*time.Minute)
	assert.Equal(t, 2*time.Minute, spacing(*rulesOf(registry)))

	registry, _ = pacedRegistry(6*time.Minute, 3*time.Minute)
	assert.Equal(t, 90*time.Second, spacing(*rulesOf(registry)))
}

func TestABoxDueBesideAQuizWaitsUntilTheQuizIsFarEnough(t *testing.T) {
	registry, clock := pacedRegistry(4*time.Minute, 4*time.Minute)
	events := playing(t, registry, "scope-a")

	until(t, registry, clock, 4*time.Minute, "scope-a")
	got := receivedOn(events)
	require.Len(t, got.quizzes, 1, "both are due: one is offered")
	require.Empty(t, got.boxes, "and the other is held")
	answerRight(t, registry, got.quizzes[0], "scope-a")

	until(t, registry, clock, 6*time.Minute-time.Second, "scope-a")
	assert.Empty(t, receivedOn(events).boxes, "the quiz is gone, but not two minutes old")

	until(t, registry, clock, 6*time.Minute, "scope-a")
	assert.Len(t, receivedOn(events).boxes, 1, "held, not redrawn: it comes the moment the quiz is far enough")
}

func TestAQuizDueBesideABoxWaitsUntilTheBoxIsFarEnough(t *testing.T) {
	registry, clock := pacedRegistry(4*time.Minute, 5*time.Minute)
	events := playing(t, registry, "scope-a")

	until(t, registry, clock, 4*time.Minute, "scope-a")
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	_, claimed := registry.Claim(boxes[0].Token, "scope-a", "scope-a")
	require.True(t, claimed)

	until(t, registry, clock, 6*time.Minute-time.Second, "scope-a")
	assert.Empty(t, receivedOn(events).quizzes, "due a minute after the box, held till two")

	until(t, registry, clock, 6*time.Minute, "scope-a")
	assert.Len(t, receivedOn(events).quizzes, 1)
}

func TestABoxWaitsForAQuizStillOutPastTheSpacing(t *testing.T) {
	registry, clock := pacedRegistry(20*time.Second, 20*time.Second)
	require.Less(t, spacing(*rulesOf(registry)), bannerFor, "the banner outlives the spacing")
	events := playing(t, registry, "scope-a")

	until(t, registry, clock, 20*time.Second, "scope-a")
	quizzes := receivedOn(events).quizzes
	require.Len(t, quizzes, 1)

	until(t, registry, clock, 35*time.Second, "scope-a")
	assert.Empty(t, receivedOn(events).boxes, "fifteen seconds is past the spacing, but the banner is still up")

	answerRight(t, registry, quizzes[0], "scope-a")
	until(t, registry, clock, 36*time.Second, "scope-a")
	assert.Len(t, receivedOn(events).boxes, 1)
}

func TestAQuizWaitsForABoxStillFlyingPastTheSpacing(t *testing.T) {
	registry, clock := pacedRegistry(20*time.Second, 25*time.Second)
	require.Less(t, spacing(*rulesOf(registry)), rulesOf(registry).OfferTTL, "the box outflies the spacing")
	events := playing(t, registry, "scope-a")

	until(t, registry, clock, 20*time.Second, "scope-a")
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)

	until(t, registry, clock, 32*time.Second, "scope-a")
	assert.Empty(t, receivedOn(events).quizzes, "due at 25s and past the spacing, but the box still flies")

	_, claimed := registry.Claim(boxes[0].Token, "scope-a", "scope-a")
	require.True(t, claimed)
	until(t, registry, clock, 33*time.Second, "scope-a")
	assert.Len(t, receivedOn(events).quizzes, 1)
}

func TestAWelcomeBoxWaitsForTheWelcomeQuizToBeGoneAndNoLonger(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	require.Greater(t, spacing(*rulesOf(registry)), time.Minute)
	entrant := EntrantOf(newcomerPayer)
	events := attend(t, registry, entrant)

	registry.Clicked(ClickOf(newcomerPayer, false))
	after(t, registry, clock, 50*time.Second)
	registry.Clicked(ClickOf(newcomerPayer, true))
	after(t, registry, clock, time.Second)
	quizzes := receivedOn(events).quizzes
	require.Len(t, quizzes, 1, "the empty bank asks at once")

	after(t, registry, clock, 10*time.Second)
	assert.Empty(t, receivedOn(events).boxes, "the minute has passed, but the quiz is still up")

	answerRight(t, registry, quizzes[0], entrant)
	after(t, registry, clock, time.Second)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1, "a welcome is not held for the spacing, only while the other is out")
	assert.Equal(t, KindSpreadClicks, boxes[0].Kind)
}
