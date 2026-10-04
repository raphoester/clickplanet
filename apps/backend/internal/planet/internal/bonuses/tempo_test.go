package bonuses

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newRegistryUnder(t *testing.T, switches *tempo.Switches) (*Registry, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(epoch)
	registry := New(Config{
		MinInterval:       10 * time.Minute,
		MaxInterval:       10 * time.Minute,
		MissRetry:         20 * time.Second,
		OfferTTL:          15 * time.Second,
		Kinds:             map[Kind]float64{KindRefill: 1},
		ActiveWithin:      30 * time.Minute,
		ForgetAfter:       30 * time.Minute,
		MaxChargesPerHour: 6,
		SweepInterval:     time.Second,
	}, clock, newFakeHoldings(), switches)

	registry.Quizzing(quizzes.Config{
		Enabled:           true,
		MinInterval:       quizWindow,
		MaxInterval:       quizWindow,
		OfferTTL:          bannerFor,
		AnswerWindow:      answerIn,
		MaxChargesPerHour: 6,
	}, fixedBank{})

	return registry, clock
}

func rulesOf(t *testing.T, boxInterval time.Duration, frozen bool) tempo.Rules {
	t.Helper()

	rules, err := tempo.NewRules(3, boxInterval, frozen)
	require.NoError(t, err)
	return rules
}

func TestABoxIntervalReplacesTheDrawnWait(t *testing.T) {
	switches := tempo.NewSwitches()
	registry, _ := newRegistryUnder(t, switches)
	switches.Set(rulesOf(t, 2*time.Minute, false))

	assert.Equal(t, 2*time.Minute, registry.window())

	switches.Set(tempo.Plain())
	assert.Equal(t, 10*time.Minute, registry.window(), "the configured window comes back")
}

func TestAShorterIntervalPullsInABoxAlreadyWaitedFor(t *testing.T) {
	switches := tempo.NewSwitches()
	registry, clock := newRegistryUnder(t, switches)
	events := playing(t, registry, "scope-a")

	switches.Set(rulesOf(t, 2*time.Minute, false))
	registry.sweep()
	clock.Advance(2*time.Minute + time.Second)
	registry.sweep()

	assert.NotNil(t, offered(t, events), "a box ten minutes away when the rules changed comes within the new interval")
}

func TestTheHourlyCapLetsThroughEveryBoxTheIntervalSends(t *testing.T) {
	switches := tempo.NewSwitches()
	registry, _ := newRegistryUnder(t, switches)

	assert.Equal(t, 6, registry.chargesPerHour())

	switches.Set(rulesOf(t, 2*time.Minute, false))
	assert.Equal(t, 30, registry.chargesPerHour())

	switches.Set(rulesOf(t, 20*time.Minute, false))
	assert.Equal(t, 6, registry.chargesPerHour(), "a slower interval never lowers the cap")
}

func TestAFrozenMapIsOfferedNoBoxAndNoQuiz(t *testing.T) {
	switches := tempo.NewSwitches()
	registry, clock := newRegistryUnder(t, switches)
	events := playing(t, registry, "scope-a")

	switches.Set(rulesOf(t, 0, true))
	clock.Advance(time.Hour)
	registry.sweep()

	assert.Nil(t, offered(t, events))
	assert.Nil(t, quizOffered(t, events))
	assert.Empty(t, registry.offers)
	assert.Empty(t, registry.quizOffers)
}

func TestABoxOfferedBeforeTheFreezeLapsesAsUsual(t *testing.T) {
	switches := tempo.NewSwitches()
	registry, clock := newRegistryUnder(t, switches)
	playing(t, registry, "scope-a")
	clock.Advance(10*time.Minute + time.Second)
	registry.sweep()
	require.Len(t, registry.offers, 1)

	switches.Set(rulesOf(t, 0, true))
	clock.Advance(time.Minute)
	registry.sweep()

	assert.Empty(t, registry.offers)
}
