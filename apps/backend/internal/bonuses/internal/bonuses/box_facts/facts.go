package box_facts

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
)

// Fact is told when it is stamped: at is when the log took it.
type Fact = func(at time.Time) proto.Message

// Facts is the offers' log of what callers did with the boxes, which relays each fact in order to whoever listens.
type Facts interface {
	Tell(fact Fact)
}

func New(facts Facts) Boxes {
	return Boxes{facts: facts}
}

type Boxes struct {
	facts Facts
}

func (b Boxes) Caught(scope string, after time.Duration) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &bonusesv1.BoxCaught{Scope: scope, After: durationpb.New(after), CaughtAt: timestamppb.New(at)}
	})
}

func (b Boxes) Lapsed(scope string) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &bonusesv1.BoxLapsed{Scope: scope, LapsedAt: timestamppb.New(at)}
	})
}

func (b Boxes) Foreign(scope string) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &bonusesv1.ForeignBoxClaimed{Scope: scope, ClaimedAt: timestamppb.New(at)}
	})
}
