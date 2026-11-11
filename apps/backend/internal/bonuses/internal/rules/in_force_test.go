package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := rules.NewInForce()
	offers := rules.Reader(inForce, bonuses.Part)

	for _, kept := range []rules.Snapshot{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
	assert.Equal(t, 30, offers().Boxes.MaxChargesPerHour)
	assert.Equal(t, time.Minute, offers().Boxes.MinInterval)
}

func TestEveryWatcherHearsOfEachSetEnforced(t *testing.T) {
	inForce := rules.NewInForce()
	first, leaveFirst := inForce.Watch()
	defer leaveFirst()
	second, leaveSecond := inForce.Watch()
	defer leaveSecond()

	inForce.Enforce(restored(t, rush()))

	assert.True(t, (<-first).Is("rush"))
	assert.True(t, (<-second).Is("rush"))
}

func TestAWatcherThatFellBehindHearsOnlyTheLatest(t *testing.T) {
	inForce := rules.NewInForce()
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
	inForce := rules.NewInForce()
	watcher, leave := inForce.Watch()
	leave()

	inForce.Enforce(restored(t, rush()))

	select {
	case set := <-watcher:
		require.Failf(t, "a watcher that left heard a set", "%v", set.TakeSnapshot().Name)
	default:
	}
}
