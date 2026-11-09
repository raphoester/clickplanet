package arenas

import (
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

const maxFreeze = 30 * time.Second

// Siege is a rule set's siege. On, a battle that plays the set is a siege, which the gamemaster reads: planet plays no
// rule of it. Freeze is how long an arena takes no play after the map goes blank and before the battle ends.
type Siege struct {
	On     bool
	Freeze time.Duration
}

func (s Siege) Validate() error {
	if s.Freeze < 0 || s.Freeze > maxFreeze {
		return fmt.Errorf("freeze is %s, outside 0s to %s", s.Freeze, maxFreeze)
	}
	return nil
}

var SiegePart = rules.PartOf[Siege]{
	Key: "siege",
	Read: func(m *planetv1.Rules) Siege {
		return Siege{On: m.GetSiege().GetOn(), Freeze: m.GetSiege().GetFreeze().AsDuration()}
	},
	InDefault: func(s Siege) error {
		if s.On {
			return errors.New("on: default never plays a siege, or every battle would be one")
		}
		return nil
	},
}
