//go:build testing

package clicks

import "fmt"

// GroundOf is a board of the borders' tiles, all at one point and none touching another, with no bridge and no
// capital: what a test of the tile map needs of a ground.
func GroundOf(borders *Borders) *Ground {
	return GroundWith(borders, NewCapitals(map[string]uint32{}))
}

// GroundWith is GroundOf, with the capitals.
func GroundWith(borders *Borders, capitals *Capitals) *Ground {
	return NamedGround("test-board", borders, capitals)
}

// NamedGround is GroundWith, under the board's name.
func NamedGround(name BoardName, borders *Borders, capitals *Capitals) *Ground {
	positions := make([]float32, borders.Tiles()*3)
	for i := 0; i < len(positions); i += 3 {
		positions[i] = 1
	}
	geography, err := NewGeography(positions, nil)
	if err != nil {
		panic(err)
	}
	bridges, err := NewBridges(borders.Tiles(), nil)
	if err != nil {
		panic(err)
	}
	ground, err := NewGround(name, geography, borders, bridges, capitals, NoProvinces())
	if err != nil {
		panic(err)
	}
	return ground
}

// OneGround is the same ground on every board.
type OneGround struct {
	Ground *Ground
}

func (o OneGround) GroundOf(BoardID) (*Ground, error) {
	return o.Ground, nil
}

// GroundsOf is a ground for each board, by its row: another board is refused.
type GroundsOf map[BoardID]*Ground

func (g GroundsOf) GroundOf(board BoardID) (*Ground, error) {
	ground, found := g[board]
	if !found {
		return nil, fmt.Errorf("board %d is not carried", board)
	}
	return ground, nil
}

// PlayedOn is a ground in play.
func PlayedOn(ground *Ground) *Played {
	played := NewPlayed()
	played.Play(ground)
	return played
}

// PlayedOf is a board of that many tiles in play: what a test that checks tile ids against the board needs.
func PlayedOf(tiles uint32) *Played {
	return PlayedOn(GroundOf(BordersOf(tiles)))
}
