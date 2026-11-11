package cprules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func inForce() *cprules.InForce[*planetv1.Rules] {
	return cprules.NewInForce[*planetv1.Rules](cptime.NewFixedClock(epoch))
}

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	rules := inForce()
	throttle := cprules.Reader(rules, clicksPart)
	perTile := cprules.Reader(rules, shieldsPart)

	for _, kept := range []snapshot{seeded(), rush()} {
		rules.Enforce(restored(t, kept))
	}

	assert.Equal(t, cprules.SetName("rush"), rules.Name())
	assert.InDelta(t, 3.0, throttle().Tempo, 1e-9)
	assert.Equal(t, uint32(60), throttle().Burst, "default's, which rush leaves unset")
	assert.Equal(t, shields(10), perTile())
}

func TestASetSaysWhenItTookOver(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	rules := cprules.NewInForce[*planetv1.Rules](clock)
	rules.Enforce(restored(t, seeded()))

	clock.Advance(time.Minute)
	rules.Enforce(restored(t, rush()))

	assert.Equal(t, epoch.Add(time.Minute), rules.Since())
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	rules := inForce()
	first, leaveFirst := rules.Watch()
	defer leaveFirst()
	second, leaveSecond := rules.Watch()
	defer leaveSecond()

	rules.Enforce(restored(t, rush()))

	assert.True(t, (<-first).Is("rush"))
	assert.True(t, (<-second).Is("rush"))
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	rules := inForce()
	watcher, leave := rules.Watch()
	defer leave()

	rules.Enforce(restored(t, rush()))
	rules.Enforce(restored(t, seeded()))

	latest := (<-watcher).TakeSnapshot()
	assert.Equal(t, cprules.Default, latest.Name)
	assert.True(t, proto.Equal(seeded().Rules, latest.Rules))
	select {
	case set := <-watcher:
		require.Failf(t, "a second set waited", "%v", set.TakeSnapshot().Name)
	default:
	}
}

func TestAWatcherThatLeftHearsNothing(t *testing.T) {
	rules := inForce()
	watcher, leave := rules.Watch()
	leave()

	rules.Enforce(restored(t, rush()))

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%v", set.TakeSnapshot().Name)
	default:
	}
}
