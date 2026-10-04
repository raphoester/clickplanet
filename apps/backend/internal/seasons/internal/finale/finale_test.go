package finale_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
)

var (
	finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
)

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
}

func TestBeforeTheFinaleTheGameIsPlainAndAfterTheEndTheMapIsFrozen(t *testing.T) {
	rules := finale.NewRules(finale.Config{})

	before := finale.PhaseAt(seasonZero(), finaleStarts.Add(-time.Second))
	assert.False(t, before.Running())
	assert.True(t, before.Switches(rules).Equal(finale.Switches{}))

	after := finale.PhaseAt(seasonZero(), seasonEnds)
	assert.False(t, after.Running())
	assert.Equal(t, calendar.Number(0), after.Season().Number)
	assert.True(t, after.Switches(rules).Frozen(), "the map freezes at the instant the season ends")
	assert.Zero(t, after.Switches(rules).RefillMultiplier())
	_, _, giving := after.Switches(rules).Gift()
	assert.False(t, giving)
}

func TestTheFinalesRulesHoldFromItsStartToTheEnd(t *testing.T) {
	rules := finale.NewRules(finale.Config{RefillMultiplier: 3, BoxInterval: 2 * time.Minute})

	for _, at := range []time.Time{finaleStarts, seasonEnds.Add(-time.Nanosecond)} {
		phase := finale.PhaseAt(seasonZero(), at)
		require.True(t, phase.Running(), at)

		switches := phase.Switches(rules)
		assert.InDelta(t, 3.0, switches.RefillMultiplier(), 1e-9)
		assert.Equal(t, 2*time.Minute, switches.BoxInterval())
		tag, madeBefore, giving := switches.Gift()
		assert.True(t, giving)
		assert.Equal(t, "season-0-finale", tag)
		assert.Equal(t, finaleStarts, madeBefore, "an account made during the finale gets nothing")
		assert.False(t, switches.Frozen())
	}
}

func TestNoSeasonIsAPlainGame(t *testing.T) {
	phase := finale.PhaseAt(calendar.New(calendar.Config{}), seasonEnds)

	assert.False(t, phase.Running())
	assert.True(t, phase.Switches(finale.NewRules(finale.Config{})).Equal(finale.Switches{}))
}

func TestTheNextSeasonUnfreezesTheMap(t *testing.T) {
	seasons := calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: seasonEnds.AddDate(0, 2, 0), Finale: 2 * time.Hour},
	}})

	phase := finale.PhaseAt(seasons, seasonEnds.Add(time.Hour))

	assert.False(t, phase.Switches(finale.NewRules(finale.Config{})).Frozen())
	assert.False(t, phase.Running())
	assert.Equal(t, calendar.Number(1), phase.Season().Number)
}

func TestTheFinaleHasItsDefaults(t *testing.T) {
	switches := finale.PhaseAt(seasonZero(), finaleStarts).Switches(finale.NewRules(finale.Config{}))

	assert.InDelta(t, 3.0, switches.RefillMultiplier(), 1e-9)
	assert.Equal(t, 2*time.Minute, switches.BoxInterval())
}

func TestABadFinaleRefusesTheBoot(t *testing.T) {
	require.NoError(t, finale.Config{}.Validate())
	require.NoError(t, finale.Config{RefillMultiplier: 1, BoxInterval: time.Minute}.Validate())

	for _, bad := range []float64{0.5, -1, math.NaN(), math.Inf(1)} {
		assert.ErrorContains(t, finale.Config{RefillMultiplier: bad}.Validate(), "finale.refillMultiplier")
	}
	assert.ErrorContains(t, finale.Config{BoxInterval: -time.Second}.Validate(), "finale.boxInterval")
}

func TestSwitchesAreEqualWhenEveryRuleIs(t *testing.T) {
	rules := finale.NewRules(finale.Config{})
	at := finale.PhaseAt(seasonZero(), finaleStarts).Switches(rules)
	later := finale.PhaseAt(seasonZero(), finaleStarts.Add(time.Hour)).Switches(rules)

	assert.True(t, at.Equal(later))
	assert.False(t, at.Equal(finale.PhaseAt(seasonZero(), seasonEnds).Switches(rules)))
}
