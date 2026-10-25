package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func TestWithNoLayerTheGamePlaysTheDefault(t *testing.T) {
	assert.Equal(t, rules.Default, rules.LayersOf(rules.Baseline{}, rules.Override{}).SetAt(at))
}

func TestWithNoOverrideTheGamePlaysTheBaseline(t *testing.T) {
	finale := rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at}.Restore()

	assert.Equal(t, rules.SetName("rush"), rules.LayersOf(finale, rules.Override{}).SetAt(at))
}

func TestAnOverridePlaysOverTheBaselineUntilItsEnd(t *testing.T) {
	days := rules.BaselineSnapshot{Request: "season-0:days", Set: rules.Default, AppliedAt: at}.Restore()
	override, err := rules.NewOverride(setNamed(t, "rush"), at, at.Add(time.Hour))
	require.NoError(t, err)
	layers := rules.LayersOf(days, override)

	assert.Equal(t, rules.Default, layers.SetAt(at.Add(-time.Second)), "before it starts")
	assert.Equal(t, rules.SetName("rush"), layers.SetAt(at))
	assert.Equal(t, rules.SetName("rush"), layers.SetAt(at.Add(time.Hour-time.Nanosecond)))
	assert.Equal(t, rules.Default, layers.SetAt(at.Add(time.Hour)), "the baseline comes back by itself")
}

func TestAnOverrideEndedNowGivesTheBaselineBack(t *testing.T) {
	finale := rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at}.Restore()
	override, err := rules.NewOverride(setNamed(t, "calm"), at, at.Add(time.Hour))
	require.NoError(t, err)

	ended := override.EndedAt(at.Add(time.Minute))

	assert.Equal(t, rules.SetName("rush"), rules.LayersOf(finale, ended).SetAt(at.Add(time.Minute)))
}
