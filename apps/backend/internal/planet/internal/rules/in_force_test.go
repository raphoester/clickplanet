package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	throttle := rules.Reader(inForce, clicks.ThrottlePart)
	toll := rules.Reader(inForce, clicks.TollPart)
	game := rules.Reader(inForce, bonuses.Part)

	for _, kept := range []rules.Snapshot{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
	assert.InDelta(t, 3.0, throttle().Tempo, 1e-9)
	assert.Len(t, toll(), 3)
	assert.Equal(t, 30, game().Boxes.MaxChargesPerHour)
}

func TestASetSaysWhenItTookOver(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	inForce := rules.NewInForce(clock)
	inForce.Enforce(restored(t, rules.Seeded()))

	clock.Advance(time.Minute)
	inForce.Enforce(restored(t, rush()))

	assert.Equal(t, epoch.Add(time.Minute), inForce.Since())
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(restored(t, rush()))

	assert.True(t, (<-first).Is("rush"))
	assert.True(t, (<-second).Is("rush"))
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	watcher, leave := inForce.Watch()
	defer leave()

	inForce.Enforce(restored(t, rush()))
	inForce.Enforce(restored(t, rules.Seeded()))

	latest := (<-watcher).TakeSnapshot()
	assert.Equal(t, rules.Default, latest.Name)
	assert.True(t, proto.Equal(rules.Seeded().Rules, latest.Rules))
	select {
	case set := <-watcher:
		require.Failf(t, "a second set waited", "%v", set.TakeSnapshot().Name)
	default:
	}
}

func TestAWatcherThatLeftHearsNothing(t *testing.T) {
	inForce := rules.NewInForce(cptime.NewFixedClock(epoch))
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(restored(t, rush()))

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%v", set.TakeSnapshot().Name)
	default:
	}
}
