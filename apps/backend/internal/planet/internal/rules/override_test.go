package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func TestAnOverrideRunsFromNowUntilItsEnd(t *testing.T) {
	override, err := rules.NewOverride(setNamed(t, "rush"), at, at.Add(30*time.Minute))
	require.NoError(t, err)

	assert.Equal(t, rules.OverrideSnapshot{Set: "rush", StartsAt: at, EndsAt: at.Add(30 * time.Minute)},
		override.TakeSnapshot())
}

func TestAnOverrideEndsAfterNowAndWithinADay(t *testing.T) {
	_, err := rules.NewOverride(setNamed(t, "rush"), at, at)
	require.ErrorIs(t, err, rules.ErrInvalidOverride)

	_, err = rules.NewOverride(setNamed(t, "rush"), at, at.Add(24*time.Hour+time.Second))
	require.ErrorIs(t, err, rules.ErrInvalidOverride, "none is left on by mistake for long")

	_, err = rules.NewOverride(setNamed(t, "rush"), at, at.Add(24*time.Hour))
	require.NoError(t, err)
}

func TestARunningOverrideEndsNow(t *testing.T) {
	override, err := rules.NewOverride(setNamed(t, "rush"), at, at.Add(time.Hour))
	require.NoError(t, err)

	ended := override.EndedAt(at.Add(10 * time.Minute))

	assert.Equal(t, rules.OverrideSnapshot{Set: "rush", StartsAt: at, EndsAt: at.Add(10 * time.Minute)},
		ended.TakeSnapshot())
}

func TestAnOverrideThatEndedAlreadyKeepsItsEnd(t *testing.T) {
	override, err := rules.NewOverride(setNamed(t, "rush"), at, at.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, override, override.EndedAt(at.Add(2*time.Hour)))
	assert.Equal(t, rules.Override{}, rules.Override{}.EndedAt(at), "no override is none")
}
