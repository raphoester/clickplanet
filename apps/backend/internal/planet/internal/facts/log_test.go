package facts_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

func opened(scope string) facts.Fact {
	return func(at time.Time) proto.Message {
		return &planetv1.StreamOpened{Scope: scope, OpenedAt: timestamppb.New(at)}
	}
}

func openedAt(t *testing.T, told []proto.Message) []time.Time {
	t.Helper()
	times := make([]time.Time, 0, len(told))
	for _, fact := range told {
		stream, ok := fact.(*planetv1.StreamOpened)
		require.True(t, ok)
		times = append(times, stream.GetOpenedAt().AsTime())
	}
	return times
}

func TestEachFactIsToldLaterThanTheOneBefore(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	log := facts.NewLog(clock)

	log.Tell(opened("a"))
	log.Tell(opened("b"))
	clock.Advance(time.Second)
	log.Tell(opened("c"))

	assert.Equal(t, []time.Time{at, at.Add(time.Nanosecond), at.Add(time.Second)}, openedAt(t, log.Take()),
		"two facts in one tick of the clock still have two times")
}

func TestAClockThatStepsBackStillTellsLater(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	log := facts.NewLog(clock)

	log.Tell(opened("a"))
	clock.Advance(-time.Minute)
	log.Tell(opened("b"))

	assert.Equal(t, []time.Time{at, at.Add(time.Nanosecond)}, openedAt(t, log.Take()))
}

func TestWhatIsTakenIsToldOnce(t *testing.T) {
	log := facts.NewLog(cptime.NewFixedClock(at))
	log.Tell(opened("a"))

	require.Len(t, log.Take(), 1)
	assert.Empty(t, log.Take())
}

func TestTheRelaySendsInTheOrderTheFactsWereToldAndOnceMoreAtShutdown(t *testing.T) {
	clock := cptime.NewFixedClock(at)
	log := facts.NewLog(clock)
	events := cpredis.NewRecordedEvents()
	relay := facts.NewRelay(log, events, time.Hour)

	for _, scope := range []string{"a", "b", "c"} {
		log.Tell(opened(scope))
		clock.Advance(time.Millisecond)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	relay.Run(ctx)

	scopes := make([]string, 0, 3)
	for _, event := range events.Published() {
		scopes = append(scopes, event.(*planetv1.StreamOpened).GetScope())
	}
	assert.Equal(t, []string{"a", "b", "c"}, scopes)
}
