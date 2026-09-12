package bonus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAnEncloseBonusSaysHowBigAShapeMayBe(t *testing.T) {
	enclosures := NewEnclosures(cptime.NewFixedClock(epoch))

	enclosures.Grant("scope-a", epoch.Add(time.Minute), 3, 10)

	maxTiles, ok := enclosures.Enclosing("scope-a")
	require.True(t, ok)
	assert.Equal(t, 10, maxTiles)
}

func TestAnEncloseBonusEndsWhenItsShapesAreSpent(t *testing.T) {
	enclosures := NewEnclosures(cptime.NewFixedClock(epoch))
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 2, 10)

	left, ok := enclosures.Spend("scope-a")
	require.True(t, ok)
	assert.Equal(t, 1, left)

	left, ok = enclosures.Spend("scope-a")
	require.True(t, ok)
	assert.Equal(t, 0, left)

	_, ok = enclosures.Spend("scope-a")
	assert.False(t, ok, "a third shape is past the bonus")

	_, ok = enclosures.Enclosing("scope-a")
	assert.False(t, ok, "a bonus with no shape left is over before its time")
}

func TestAnEncloseBonusEndsWithItsTime(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	enclosures := NewEnclosures(clock)
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 3, 10)

	clock.Advance(time.Minute)

	_, ok := enclosures.Enclosing("scope-a")
	assert.False(t, ok, "the end is not part of the bonus")

	_, ok = enclosures.Spend("scope-a")
	assert.False(t, ok)
}

func TestAnEncloseBonusBelongsToTheCallerWhoCaughtIt(t *testing.T) {
	enclosures := NewEnclosures(cptime.NewFixedClock(epoch))
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 3, 10)

	_, ok := enclosures.Enclosing("scope-b")
	assert.False(t, ok)

	_, ok = enclosures.Spend("scope-b")
	assert.False(t, ok)
}

func TestAGrantForgetsTheEnclosuresThatAreOver(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	enclosures := NewEnclosures(clock)

	enclosures.Grant("timed-out", epoch.Add(time.Minute), 3, 10)
	enclosures.Grant("spent", epoch.Add(time.Hour), 1, 10)
	_, _ = enclosures.Spend("spent")

	clock.Advance(2 * time.Minute)
	enclosures.Grant("scope-c", clock.Now().Add(time.Minute), 3, 10)

	assert.Len(t, enclosures.running, 1, "a bonus that is over is not kept forever")
}
