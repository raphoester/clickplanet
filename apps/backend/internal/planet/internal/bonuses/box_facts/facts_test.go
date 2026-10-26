package box_facts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/box_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestEachBoxIsToldWithTheScopeItWasOfferedTo(t *testing.T) {
	at := time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)
	log := facts.NewLog(cptime.NewFixedClock(at))
	boxes := box_facts.New(log)

	boxes.Caught("1.2.3.4", 800*time.Millisecond)
	boxes.Lapsed("1.2.3.4")
	boxes.Foreign("5.6.7.8")

	told := log.Take()
	require.Len(t, told, 3)
	for i, want := range []proto.Message{
		&planetv1.BoxCaught{Scope: "1.2.3.4", After: durationpb.New(800 * time.Millisecond), CaughtAt: timestamppb.New(at)},
		&planetv1.BoxLapsed{Scope: "1.2.3.4", LapsedAt: timestamppb.New(at.Add(time.Nanosecond))},
		&planetv1.ForeignBoxClaimed{Scope: "5.6.7.8", ClaimedAt: timestamppb.New(at.Add(2 * time.Nanosecond))},
	} {
		assert.True(t, proto.Equal(want, told[i]), "fact %d: got %v", i, told[i])
	}
}
