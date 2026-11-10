package arenas

import (
	"errors"
	"fmt"
	"math"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

const (
	maxFreeze = 30 * time.Second

	mostShieldsOnATile = math.MaxUint8
)

// Rules is a rule set's arena. On, a battle that plays the set is an arena battle, fought on a blank map: the gamemaster
// reads it, and asks planet to open and close the arena. Freeze is how long an arena takes no play after the map goes
// blank and before the battle ends. CapitalShields is what each capital starts the arena with.
type Rules struct {
	On             bool
	Freeze         time.Duration
	CapitalShields int
}

func (r Rules) Validate() error {
	var freeze error
	if r.Freeze < 0 || r.Freeze > maxFreeze {
		freeze = fmt.Errorf("freeze is %s, outside 0s to %s", r.Freeze, maxFreeze)
	}
	return errors.Join(freeze, rules.AtMost("capitalShields", r.CapitalShields, "what a tile keeps", mostShieldsOnATile))
}

var Part = rules.PartOf[Rules]{
	Key: "arena",
	Read: func(m *planetv1.Rules) Rules {
		return Rules{
			On:             m.GetArena().GetOn(),
			Freeze:         m.GetArena().GetFreeze().AsDuration(),
			CapitalShields: int(m.GetArena().GetCapitalShields()),
		}
	},
	InDefault: func(r Rules) error {
		if r.On {
			return errors.New("on: default never plays in an arena, or every battle would")
		}
		return nil
	},
}
