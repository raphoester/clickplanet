package bonus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestABombCanBeDroppedOnce(t *testing.T) {
	bombs := NewBombs(cptime.NewFixedClock(epoch))

	bombs.Grant("scope-a", epoch.Add(time.Minute))

	assert.True(t, bombs.Take("scope-a"))
	assert.False(t, bombs.Take("scope-a"), "a bomb that went off is gone")
}

func TestABombHeldTooLongIsLost(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	bombs := NewBombs(clock)

	bombs.Grant("scope-a", epoch.Add(30*time.Second))
	clock.Advance(30 * time.Second)

	assert.False(t, bombs.Take("scope-a"))
}

func TestABombBelongsToTheCallerWhoCaughtIt(t *testing.T) {
	bombs := NewBombs(cptime.NewFixedClock(epoch))

	bombs.Grant("scope-a", epoch.Add(time.Minute))

	assert.False(t, bombs.Take("scope-b"))
	assert.True(t, bombs.Take("scope-a"))
}

func TestAGrantForgetsTheBombsThatLapsed(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	bombs := NewBombs(clock)

	bombs.Grant("scope-a", epoch.Add(time.Minute))
	clock.Advance(2 * time.Minute)
	bombs.Grant("scope-b", clock.Now().Add(time.Minute))

	assert.Len(t, bombs.until, 1)
}
