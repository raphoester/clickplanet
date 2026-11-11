package box_facts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/box_facts"
)

// log stamps each fact a nanosecond after the one before, as the offers' facts.Log does.
type log struct {
	at   time.Time
	told []proto.Message
}

func (l *log) Tell(fact box_facts.Fact) {
	l.told = append(l.told, fact(l.at))
	l.at = l.at.Add(time.Nanosecond)
}

func TestEachBoxIsToldWithTheScopeItWasOfferedTo(t *testing.T) {
	at := time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)
	facts := &log{at: at}
	boxes := box_facts.New(facts)

	boxes.Caught("1.2.3.4", 800*time.Millisecond)
	boxes.Lapsed("1.2.3.4")
	boxes.Foreign("5.6.7.8")

	require.Len(t, facts.told, 3)
	for i, want := range []proto.Message{
		&bonusesv1.BoxCaught{Scope: "1.2.3.4", After: durationpb.New(800 * time.Millisecond), CaughtAt: timestamppb.New(at)},
		&bonusesv1.BoxLapsed{Scope: "1.2.3.4", LapsedAt: timestamppb.New(at.Add(time.Nanosecond))},
		&bonusesv1.ForeignBoxClaimed{Scope: "5.6.7.8", ClaimedAt: timestamppb.New(at.Add(2 * time.Nanosecond))},
	} {
		assert.True(t, proto.Equal(want, facts.told[i]), "fact %d: got %v", i, facts.told[i])
	}
}
