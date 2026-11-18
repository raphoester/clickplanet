package arenas

import (
	"errors"
	"fmt"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

const maxFreeze = 30 * time.Second

// Rules is a rule set's arena. On, a battle that plays the set is an arena battle, fought on a fresh map: the gamemaster
// reads it, and asks planet to open and close the arena. Freeze is how long an arena takes no play after its map plays
// and before the battle ends. Board is the board the fresh map plays: empty, the world's.
type Rules struct {
	On     bool
	Freeze time.Duration
	Board  clicks.BoardName
}

func (r Rules) Validate() error {
	if r.Freeze < 0 || r.Freeze > maxFreeze {
		return fmt.Errorf("freeze is %s, outside 0s to %s", r.Freeze, maxFreeze)
	}
	if r.Board != "" {
		if _, err := cpboards.Named(cpboards.Files(), string(r.Board)); err != nil {
			return fmt.Errorf("board: %w", err)
		}
	}
	return nil
}

var Part = cprules.PartOf[*planetv1.Rules, Rules]{
	Key: "arena",
	Read: func(m *planetv1.Rules) Rules {
		return Rules{
			On: m.GetArena().GetOn(), Freeze: m.GetArena().GetFreeze().AsDuration(),
			Board: clicks.BoardName(m.GetArena().GetBoard()),
		}
	},
	InDefault: func(r Rules) error {
		if r.On {
			return errors.New("on: default never plays in an arena, or every battle would")
		}
		if r.Board != "" {
			return errors.New("board: default names none, so an arena plays the world's board unless its set names one")
		}
		return nil
	},
}
