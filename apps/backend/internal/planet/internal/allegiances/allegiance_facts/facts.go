package allegiance_facts

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
)

type Facts interface {
	Tell(fact facts.Fact)
}

func New(facts Facts) Pledges {
	return Pledges{facts: facts}
}

type Pledges struct {
	facts Facts
}

var _ allegiances.Teller = Pledges{}

func (p Pledges) Pledged(account allegiances.Account, flag string, previous string) {
	p.facts.Tell(func(at time.Time) proto.Message {
		return &planetv1.AllegiancePledged{
			AccountId: string(account), Country: flag, Previous: previous, PledgedAt: timestamppb.New(at),
		}
	})
}
