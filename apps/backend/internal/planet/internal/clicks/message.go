package clicks

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var ThrottlePart = cprules.PartOf[*planetv1.Rules, Throttle]{
	Key: "clicks", Read: func(m *planetv1.Rules) Throttle { return ThrottleOf(m.GetClicks()) },
}

var TollPart = cprules.PartOf[*planetv1.Rules, TollSteps]{
	Key: "toll", Read: func(m *planetv1.Rules) TollSteps { return TollOf(m.GetToll()) },
}

func ThrottleOf(m *planetv1.ClickRules) Throttle {
	return Throttle{
		Tempo:                m.GetTempo(),
		PerSecond:            m.GetPerSecond(),
		Burst:                int(m.GetBurst()),
		ScopeMultiplier:      m.GetScopeMultiplier(),
		V6ScopeMultiplier:    m.GetV6ScopeMultiplier(),
		GuestScopeMultiplier: m.GetGuestScopeMultiplier(),
		LinkedMultiplier:     m.GetLinkedMultiplier(),
		NewAccountClicks:     m.GetNewAccountClicks(),
		FastStart:            m.GetFastStart().AsDuration(),
	}
}

func TollOf(m *planetv1.TollRules) TollSteps {
	steps := make(TollSteps, len(m.GetSteps()))
	for i, step := range m.GetSteps() {
		steps[i] = TollStep{Share: step.GetShare(), Slowdown: step.GetSlowdown()}
	}

	return steps
}
