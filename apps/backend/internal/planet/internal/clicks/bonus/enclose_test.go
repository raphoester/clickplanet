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

	enclosure, ok := enclosures.Running("scope-a")
	require.True(t, ok)
	assert.Equal(t, 10, enclosure.MaxTiles())
}

func TestAnEncloseBonusEndsWhenItsShapesAreSpent(t *testing.T) {
	enclosures := NewEnclosures(cptime.NewFixedClock(epoch))
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 2, 10)
	enclosure, _ := enclosures.Running("scope-a")

	left, ok := enclosure.Spend()
	require.True(t, ok)
	assert.Equal(t, 1, left)

	left, ok = enclosure.Spend()
	require.True(t, ok)
	assert.Equal(t, 0, left)

	_, ok = enclosure.Spend()
	assert.False(t, ok, "a third shape is past the bonus")

	_, ok = enclosures.Running("scope-a")
	assert.False(t, ok, "a bonus with no shape left is over before its time")
}

func TestAnEncloseBonusEndsWithItsTime(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	enclosures := NewEnclosures(clock)
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 3, 10)
	enclosure, _ := enclosures.Running("scope-a")

	clock.Advance(time.Minute)

	_, ok := enclosures.Running("scope-a")
	assert.False(t, ok, "the end is not part of the bonus")

	_, ok = enclosure.Spend()
	assert.False(t, ok, "a bonus picked up before its end cannot be spent after it")
}

func TestAnEncloseBonusBelongsToTheCallerWhoCaughtIt(t *testing.T) {
	enclosures := NewEnclosures(cptime.NewFixedClock(epoch))
	enclosures.Grant("scope-a", epoch.Add(time.Minute), 3, 10)

	_, ok := enclosures.Running("scope-b")
	assert.False(t, ok)
}

func TestAGrantForgetsTheEnclosuresThatAreOver(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	enclosures := NewEnclosures(clock)

	enclosures.Grant("timed-out", epoch.Add(time.Minute), 3, 10)
	enclosures.Grant("spent", epoch.Add(time.Hour), 1, 10)
	spent, _ := enclosures.Running("spent")
	_, _ = spent.Spend()

	clock.Advance(2 * time.Minute)
	enclosures.Grant("scope-c", clock.Now().Add(time.Minute), 3, 10)

	assert.Len(t, enclosures.running, 1, "a bonus that is over is not kept forever")
}
