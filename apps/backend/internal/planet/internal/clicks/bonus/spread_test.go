package bonus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestASpreadRunsUntilItsEndAndNotAfter(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	spreads := NewSpreads(clock)

	spreads.Grant("scope-a", epoch.Add(time.Minute))
	assert.True(t, spreads.Spreading("scope-a"))

	clock.Advance(time.Minute)
	assert.False(t, spreads.Spreading("scope-a"), "the end is not part of the bonus")
}

func TestASpreadBelongsToTheCallerWhoCaughtIt(t *testing.T) {
	spreads := NewSpreads(cptime.NewFixedClock(epoch))

	spreads.Grant("scope-a", epoch.Add(time.Minute))

	assert.False(t, spreads.Spreading("scope-b"))
}

func TestAGrantForgetsTheSpreadsThatRanOut(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	spreads := NewSpreads(clock)

	spreads.Grant("scope-a", epoch.Add(time.Minute))
	clock.Advance(2 * time.Minute)
	spreads.Grant("scope-b", clock.Now().Add(time.Minute))

	assert.Len(t, spreads.until, 1, "a lapsed spread is not kept forever")
}
