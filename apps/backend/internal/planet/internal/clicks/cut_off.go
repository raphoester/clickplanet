package clicks

import (
	"errors"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var ErrNoCutOff = errors.New("the takes cut none of the flag's land from its capital")

// CutOff is a flag's land that takes cut from its capital: every tile of it went blank.
type CutOff struct {
	Country string
	By      string
	// The tile whose take cut it: the flag's last tile the write took.
	Tile    uint32
	Cleared []uint32
}

// Losses is the tiles one write took from a flag, in order, and the flag that took the last of them.
type Losses struct {
	Flag  string
	Tiles []uint32
	By    string
}

// Links is what holds a flag's land together: tiles next to each other, and the two ends of a bridge.
type Links interface {
	Neighbourhood
	Crossings
}

// CutOffError is nil while the cut-off plays for a flag with a capital: a flag with none has no land to cut from it.
func (r CapitalRules) CutOffError(capital bool) error {
	if !r.CutOff || !capital {
		return ErrNoCutOff
	}
	return nil
}

// stranded is the flag's land that no path over its own tiles and the bridges it holds both ends of leads to from its
// capital, now that the lost tiles left it: only the land next to them, so land cut off before stays. Every tile next
// to them is stranded when the capital is not the flag's.
func stranded(links Links, capital uint32, holds func(tile uint32) bool, lost []uint32) []uint32 {
	edge := edgeOf(links, holds, lost)
	if len(edge) == 0 || (holds(capital) && together(links, edge)) {
		return nil
	}

	reached := reachedFrom(links, capital, holds, edge)
	var cut []uint32
	for _, tile := range edge {
		if !reached.Contains(tile) {
			cut = append(cut, flood(links, tile, holds, reached)...)
		}
	}
	slices.Sort(cut)
	return cut
}

// edgeOf is the flag's tiles linked to a lost tile: every path of its land that went through the lost tiles enters and
// leaves them on its edge.
func edgeOf(links Links, holds func(tile uint32) bool, lost []uint32) []uint32 {
	seen := cpcolls.NewSet[uint32]()
	var edge []uint32
	for _, tile := range lost {
		eachLinked(links, tile, func(next uint32) {
			if !seen.Contains(next) && holds(next) {
				seen.Add(next)
				edge = append(edge, next)
			}
		})
	}
	return edge
}

// together is true when the edge's tiles link to each other with no other tile: no path through the lost tiles is lost
// then, as one of them still reaches the capital.
func together(links Links, edge []uint32) bool {
	in := cpcolls.NewSet(edge...)
	return len(flood(links, edge[0], in.Contains, cpcolls.NewSet[uint32]())) == len(edge)
}

// reachedFrom is the land a walk from the capital reaches: it stops once every edge tile is reached, or else covers the
// capital's whole land.
func reachedFrom(links Links, capital uint32, holds func(tile uint32) bool, edge []uint32) *cpcolls.Set[uint32] {
	reached := cpcolls.NewSet[uint32]()
	if !holds(capital) {
		return reached
	}
	wanted := cpcolls.NewSet(edge...)

	left := wanted.Len()
	queue := []uint32{capital}
	reached.Add(capital)
	for len(queue) > 0 && left > 0 {
		tile := queue[0]
		queue = queue[1:]
		if wanted.Contains(tile) {
			left--
		}
		eachLinked(links, tile, func(next uint32) {
			if !reached.Contains(next) && holds(next) {
				reached.Add(next)
				queue = append(queue, next)
			}
		})
	}
	return reached
}

// flood is the land linked to the tile that no walk reached yet, marked reached as it goes.
func flood(links Links, from uint32, holds func(tile uint32) bool, reached *cpcolls.Set[uint32]) []uint32 {
	reached.Add(from)
	land := []uint32{from}
	for i := 0; i < len(land); i++ {
		eachLinked(links, land[i], func(next uint32) {
			if !reached.Contains(next) && holds(next) {
				reached.Add(next)
				land = append(land, next)
			}
		})
	}
	return land
}

func eachLinked(links Links, tile uint32, see func(next uint32)) {
	for _, next := range links.Neighbours(tile) {
		see(next)
	}
	for _, next := range links.Across(tile) {
		see(next)
	}
}

// A take, and the flag it was for.
type take struct {
	impact *Impact
	by     string
}

func takesOf(flag string, impacts []Impact) []take {
	takes := make([]take, len(impacts))
	for i := range impacts {
		takes[i] = take{impact: &impacts[i], by: flag}
	}
	return takes
}

// cutOffAll asks, once every take of a write and its knock-outs are written, what land of each flag the takes took
// tiles from is cut from its capital: one ask per flag, so tiles lost together are read together. The cut-off is told on
// the flag's last take.
func cutOffAll(tiles Claimable, takes []take) error {
	var order []string
	lost := map[string]*Losses{}
	last := map[string]*Impact{}
	for _, each := range takes {
		from := each.impact.Owner
		if each.impact.Outcome != Taken || from == "" {
			continue
		}
		losses, ok := lost[from]
		if !ok {
			losses = &Losses{Flag: from}
			lost[from] = losses
			order = append(order, from)
		}
		losses.Tiles = append(losses.Tiles, each.impact.Tile)
		losses.By = each.by
		last[from] = each.impact
	}

	for _, flag := range order {
		cutOff, err := tiles.CutOff(*lost[flag])
		switch {
		case err == nil:
			last[flag].CutOff = &cutOff
		case errors.Is(err, ErrNoCutOff):
		default:
			return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
	}
	return nil
}
