package box_facts

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
)

type Facts interface {
	Tell(fact facts.Fact)
}

func New(facts Facts) Boxes {
	return Boxes{facts: facts}
}

type Boxes struct {
	facts Facts
}

func (b Boxes) Caught(scope string, after time.Duration) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &planetv1.BoxCaught{Scope: scope, After: durationpb.New(after), CaughtAt: timestamppb.New(at)}
	})
}

func (b Boxes) Lapsed(scope string) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &planetv1.BoxLapsed{Scope: scope, LapsedAt: timestamppb.New(at)}
	})
}

func (b Boxes) Foreign(scope string) {
	b.facts.Tell(func(at time.Time) proto.Message {
		return &planetv1.ForeignBoxClaimed{Scope: scope, ClaimedAt: timestamppb.New(at)}
	})
}
