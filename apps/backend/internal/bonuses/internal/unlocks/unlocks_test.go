package unlocks_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
)

var monday = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func TestALaterDecisionReplacesTheKindsWhole(t *testing.T) {
	kept := unlocks.Unlocks{}.
		Decided(unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb}, monday)).
		Decided(unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindEncloseClicks}, monday.Add(time.Hour)))

	assert.Equal(t, unlocks.Snapshot{Kinds: bonuses.Unlocked{bonuses.KindEncloseClicks}, DecidedAt: monday.Add(time.Hour)},
		kept.TakeSnapshot(), "fewer kinds than before are fewer")
}

func TestAnOlderOrEqualDecisionChangesNothing(t *testing.T) {
	kept := unlocks.Unlocks{}.Decided(unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindBomb}, monday))

	for _, late := range []time.Time{monday, monday.Add(-time.Minute)} {
		assert.Equal(t, kept, kept.Decided(unlocks.DecisionOf(nil, late)))
	}
}
