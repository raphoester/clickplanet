package charges

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrBombsOff is a drop while the rules in force hold no bomb: the bomb stays held.
var ErrBombsOff = errors.New("bombs are off: the bomb stays held")

// Spaced is the board a bomb lands on: its tiles' spacing.
type Spaced interface {
	Spacing() float64
}

// NewBombing measures a blast in the spacings of the board in play.
func NewBombing(rules func() Rules, board Spaced) Bombing {
	return Bombing{rules: rules, board: board}
}

type Bombing struct {
	rules func() Rules
	board Spaced
}

// Refuses is ErrBombsOff while the rules in force hold bombs off.
func (b Bombing) Refuses() error {
	if b.rules().Bomb.Off {
		return ErrBombsOff
	}
	return nil
}

func (b Bombing) Radius() float64 {
	return b.rules().Bomb.Rings * b.board.Spacing()
}

type Ground interface {
	Position(id uint32) (clicks.Vec3, bool)
	Within(centre clicks.Vec3, radius float64) []uint32
}

func (b Bombing) Blast(country string, target clicks.Vec3, nearest uint32, arc float64, ground Ground) clicks.Blast {
	blast := clicks.Blast{CountryID: country, Radius: b.Radius(), Point: target.Unit()}
	if arc > b.board.Spacing() {
		return blast
	}

	blast.Tile = nearest
	blast.Point, _ = ground.Position(nearest)
	blast.Cleared = ground.Within(blast.Point, blast.Radius)

	return blast
}
