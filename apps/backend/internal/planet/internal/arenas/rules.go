package arenas

import (
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

const maxFreeze = 30 * time.Second

// Rules is a rule set's arena. On, a battle that plays the set is an arena battle, fought on a fresh map: the gamemaster
// reads it, and asks planet to open and close the arena. Freeze is how long an arena takes no play after its map plays
// and before the battle ends.
type Rules struct {
	On     bool
	Freeze time.Duration
}

func (r Rules) Validate() error {
	if r.Freeze < 0 || r.Freeze > maxFreeze {
		return fmt.Errorf("freeze is %s, outside 0s to %s", r.Freeze, maxFreeze)
	}
	return nil
}

var Part = rules.PartOf[Rules]{
	Key: "arena",
	Read: func(m *planetv1.Rules) Rules {
		return Rules{On: m.GetArena().GetOn(), Freeze: m.GetArena().GetFreeze().AsDuration()}
	},
	InDefault: func(r Rules) error {
		if r.On {
			return errors.New("on: default never plays in an arena, or every battle would")
		}
		return nil
	},
}
