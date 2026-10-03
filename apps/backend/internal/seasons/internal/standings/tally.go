package standings

import (
	"bytes"
	"maps"
	"math"
)

type Tally struct {
	Main  Country
	Tiles map[Country]uint64
}

func (t Tally) WithTake(country Country) Tally {
	tiles := maps.Clone(t.Tiles)
	if tiles == nil {
		tiles = map[Country]uint64{}
	}
	tiles[country]++

	main := t.Main
	if t.Empty() || tiles[country] > tiles[main] {
		main = country
	}
	return Tally{Main: main, Tiles: tiles}
}

func (t Tally) Empty() bool {
	return t.Main == ""
}

func (t Tally) LineOf(account AccountID) Line {
	return Line{Account: account, Country: t.Main, Tiles: t.Tiles[t.Main]}
}

type Line struct {
	Account AccountID
	Country Country
	Tiles   uint64
}

func (l Line) Above(other Line) bool {
	if l.Tiles != other.Tiles {
		return l.Tiles > other.Tiles
	}
	return bytes.Compare(l.Account[:], other.Account[:]) < 0
}

func (l Line) Cursor() Cursor {
	return Cursor{Tiles: l.Tiles, Account: l.Account}
}

type Cursor struct {
	Tiles   uint64
	Account AccountID
}

var Start = Cursor{Tiles: math.MaxInt64}

func (c Cursor) Before(line Line) bool {
	return Line{Account: c.Account, Tiles: c.Tiles}.Above(line)
}
