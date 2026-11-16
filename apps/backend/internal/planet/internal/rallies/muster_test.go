package rallies_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestARallyIsMadeByTheRulesInForceAsItIsCalled(t *testing.T) {
	rules := charges.TestRules()
	muster := rallies.NewMuster(func() charges.Rules { return rules }, 0.004, &rallies.SequentialIDs{},
		cptime.NewFixedClock(calledAt))

	rally, err := muster.Rally(alice, "fr", clicks.Vec3{Z: 1})
	require.NoError(t, err)

	kept := rally.TakeSnapshot()
	assert.Equal(t, "fr", kept.Flag)
	assert.Equal(t, clicks.Vec3{Z: 1}, kept.Centre)
	assert.Equal(t, alice, kept.Account)
	assert.InDelta(t, 18*0.004, kept.Radius, 1e-12, "18 tile spacings")
	assert.Equal(t, calledAt, kept.CalledAt)
	assert.Equal(t, calledAt.Add(3*time.Minute), kept.EndsAt)
	assert.InDelta(t, 18*0.004, muster.Radius(), 1e-12)

	rules.Rally.Rings, rules.Rally.Lasts = 10, time.Minute
	assert.InDelta(t, 3*time.Minute, kept.EndsAt.Sub(kept.CalledAt), 0, "a rally keeps what it was called with")
	again, err := muster.Rally(alice, "fr", clicks.Vec3{Z: 1})
	require.NoError(t, err)
	assert.InDelta(t, 10*0.004, again.TakeSnapshot().Radius, 1e-12, "the next one reads the rules again")
	assert.NotEqual(t, kept.ID, again.TakeSnapshot().ID)
}
