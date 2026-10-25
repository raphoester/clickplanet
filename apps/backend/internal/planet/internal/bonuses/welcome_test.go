package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ordinaryWindow = 5 * time.Minute
	newFor         = 10 * time.Minute
	firstBoxAfter  = time.Minute
)

var (
	newcomerPayer = clicks.Payer{Scope: "scope-n", Account: "newcomer", Created: epoch}
	veteranPayer  = clicks.Payer{Scope: "scope-v", Account: "veteran", Created: epoch.Add(-24 * time.Hour)}
	everyWeight   = map[Kind]float64{KindRefill: 5, KindSpreadClicks: 3, KindEncloseClicks: 2, KindBomb: 1, KindShields: 3}
)

func newWelcomingRegistry(t *testing.T, kinds map[Kind]float64, welcome WelcomeConfig) (*Registry, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		MissRetry:     20 * time.Second,
		OfferTTL:      15 * time.Second,
		Kinds:         kinds,
		Welcome:       welcome,
		ActiveWithin:  5 * time.Minute,
		ForgetAfter:   5 * time.Minute,
		SweepInterval: time.Second,
	}, clock, NewMemoryStore())
	registry.SetPace(Pace{MinInterval: ordinaryWindow, MaxInterval: ordinaryWindow, MaxChargesPerHour: 6})

	registry.Quizzing(quizzes.Config{
		Enabled:           true,
		MinInterval:       ordinaryWindow,
		MaxInterval:       ordinaryWindow,
		MissRetry:         quizRetry,
		OfferTTL:          bannerFor,
		AnswerWindow:      answerIn,
		MaxChargesPerHour: 6,
	}, fixedBank{})

	return registry, clock
}

func welcoming(t *testing.T, kinds map[Kind]float64) (*Registry, *cptime.FixedClock) {
	t.Helper()

	return newWelcomingRegistry(t, kinds, WelcomeConfig{Within: newFor, BoxAfter: firstBoxAfter})
}

type received struct {
	boxes   []*Offer
	quizzes []*QuizOffer
}

func receivedOn(events <-chan Event) received {
	var got received
	for {
		select {
		case event := <-events:
			if event.Offer != nil {
				got.boxes = append(got.boxes, event.Offer)
			}
			if event.Quiz != nil {
				got.quizzes = append(got.quizzes, event.Quiz)
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

func answerRight(t *testing.T, r *Registry, quiz *QuizOffer, entrant Entrant) Answered {
	t.Helper()

	asked, ok := r.OpenQuiz(quiz.Token, entrant)
	require.True(t, ok)

	answered, ok := r.AnswerQuiz(quiz.Token, entrant, indexOf(asked.Options, rightAnswer))
	require.True(t, ok)
	require.True(t, answered.Correct)

	return answered
}

func TestANewcomersFirstBoxIsASpreadAMinuteAfterItsAccountWasMade(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, EntrantOf(newcomerPayer))
	registry.Clicked(ClickOf(newcomerPayer, false))

	after(t, registry, clock, firstBoxAfter-time.Second)
	require.Empty(t, receivedOn(events).boxes, "not before the minute")

	after(t, registry, clock, time.Second)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	assert.Equal(t, KindSpreadClicks, boxes[0].Kind)
}

func TestTheFirstMinuteCountsFromTheAccountNotFromItsLaterClick(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, EntrantOf(newcomerPayer))

	clock.Advance(3 * time.Minute)
	registry.Clicked(ClickOf(newcomerPayer, false))
	registry.sweep(t.Context())

	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1, "a minute after it was made has passed already")
	assert.Equal(t, KindSpreadClicks, boxes[0].Kind)
}

func TestANewcomerIsWelcomedWithOneBoxOnly(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	entrant := EntrantOf(newcomerPayer)
	events := attend(t, registry, entrant)
	registry.Clicked(ClickOf(newcomerPayer, false))

	after(t, registry, clock, firstBoxAfter)
	boxes := receivedOn(events).boxes
	require.Len(t, boxes, 1)
	_, claimed := registry.Claim(boxes[0].Token, entrant, newcomerPayer.Scope)
	require.True(t, claimed)

	registry.Clicked(ClickOf(newcomerPayer, false))
	after(t, registry, clock, ordinaryWindow-time.Second)
	assert.Empty(t, receivedOn(events).boxes, "the next box waits the ordinary window after the claim")

	registry.Clicked(ClickOf(newcomerPayer, false))
	after(t, registry, clock, time.Second)
	assert.Len(t, receivedOn(events).boxes, 1)
}

func TestAnOlderAccountWaitsTheOrdinaryWindow(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, EntrantOf(veteranPayer))
	registry.Clicked(ClickOf(veteranPayer, false))

	after(t, registry, clock, firstBoxAfter)
	assert.Empty(t, receivedOn(events).boxes)

	registry.Clicked(ClickOf(veteranPayer, false))
	after(t, registry, clock, ordinaryWindow-firstBoxAfter)
	assert.Len(t, receivedOn(events).boxes, 1)
}

func TestAnAccountThatSaysNothingOfItsAgeIsNotWelcomed(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	unknown := clicks.Payer{Scope: "scope-u", Account: "unknown"}
	events := attend(t, registry, EntrantOf(unknown))
	registry.Clicked(ClickOf(unknown, true))

	after(t, registry, clock, firstBoxAfter)

	got := receivedOn(events)
	assert.Empty(t, got.boxes)
	assert.Empty(t, got.quizzes)
}

func TestWithNoWelcomeANewcomerWaitsLikeAnybody(t *testing.T) {
	registry, clock := newWelcomingRegistry(t, everyWeight, WelcomeConfig{})
	events := attend(t, registry, EntrantOf(newcomerPayer))
	registry.Clicked(ClickOf(newcomerPayer, true))

	after(t, registry, clock, firstBoxAfter)

	got := receivedOn(events)
	assert.Empty(t, got.boxes)
	assert.Empty(t, got.quizzes)
}

func TestAWelcomeKindTheConfigLeavesOutGivesNoEarlyBoxAndNoEarlyRefill(t *testing.T) {
	registry, clock := welcoming(t, map[Kind]float64{KindRefill: 1})
	events := attend(t, registry, EntrantOf(newcomerPayer))
	registry.Clicked(ClickOf(newcomerPayer, true))

	after(t, registry, clock, firstBoxAfter)

	got := receivedOn(events)
	assert.Empty(t, got.boxes, "a fresh account is never handed a bank early")
	assert.Empty(t, got.quizzes)

	registry.Clicked(ClickOf(newcomerPayer, false))
	after(t, registry, clock, ordinaryWindow)
	assert.Len(t, receivedOn(events).boxes, 1, "the ordinary schedule goes on")
}

func TestANewcomerWhoseClickEmptiesItsBankIsAskedAQuizForShieldsAtOnce(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	entrant := EntrantOf(newcomerPayer)
	events := attend(t, registry, entrant)

	registry.Clicked(ClickOf(newcomerPayer, false))
	after(t, registry, clock, time.Second)
	require.Empty(t, receivedOn(events).quizzes, "clicks still in the bank")

	registry.Clicked(ClickOf(newcomerPayer, true))
	after(t, registry, clock, time.Second)

	quizzes := receivedOn(events).quizzes
	require.Len(t, quizzes, 1)
	assert.Equal(t, KindShields, answerRight(t, registry, quizzes[0], entrant).Reward.Kind)
}

func TestOnlyTheFirstEmptyBankAsksTheWelcomeQuiz(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	entrant := EntrantOf(newcomerPayer)
	events := attend(t, registry, entrant)

	registry.Clicked(ClickOf(newcomerPayer, true))
	after(t, registry, clock, time.Second)
	quizzes := receivedOn(events).quizzes
	require.Len(t, quizzes, 1)
	answerRight(t, registry, quizzes[0], entrant)

	registry.Clicked(ClickOf(newcomerPayer, true))
	after(t, registry, clock, time.Second)
	assert.Empty(t, receivedOn(events).quizzes)
}

func TestAnOlderAccountWhoseBankRunsDryIsAskedNothing(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, EntrantOf(veteranPayer))

	registry.Clicked(ClickOf(veteranPayer, true))
	after(t, registry, clock, time.Second)

	assert.Empty(t, receivedOn(events).quizzes)
}

func TestANewcomerHoldingAFullStackOfShieldsIsAskedNoWelcomeQuiz(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	events := attend(t, registry, EntrantOf(newcomerPayer))
	holdingsOf(registry).Keep(HolderOf(newcomerPayer), Held{Shields: registry.charges.Shields})

	registry.Clicked(ClickOf(newcomerPayer, true))
	after(t, registry, clock, time.Second)

	assert.Empty(t, receivedOn(events).quizzes)
}

func TestANewcomerIsForgottenOnceItIsNoLongerNew(t *testing.T) {
	registry, clock := welcoming(t, everyWeight)
	attend(t, registry, EntrantOf(newcomerPayer))
	registry.Clicked(ClickOf(newcomerPayer, false))
	require.Len(t, registry.newcomers, 1)

	after(t, registry, clock, newFor)

	assert.Empty(t, registry.newcomers)
}
