package clicks

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrCarryover = errors.New("a carry-over that does not fit its boards")
	// ErrNotInPlay is a switch of a map that does not play: an arena plays in its place.
	ErrNotInPlay = errors.New("the map does not play: an arena plays in its place")
	// ErrPlaysAlready is a switch onto the board the map plays.
	ErrPlaysAlready = errors.New("the map plays that board already")
)

// Carryover is where each tile of a new board comes from on an old one, its parent: the old tile whose cell holds it,
// or on a coast the nearest old tile within about 0.6 of the old spacing, or none, so a new coast or island starts
// blank (docs/boards.md, "The world's switch"). An old tile's children are the new tiles whose parent it is, in id
// order; a coast the new board draws as sea has none.
type Carryover struct {
	from, to BoardName
	// parents[child-1] is the child's parent, 0 for none.
	parents []uint32
	// The children of parent p are children[at[p]:at[p+1]], ascending.
	at       []uint32
	children []uint32
}

// NewCarryover reads one parent per tile of the new board, each a tile of the old one or 0.
func NewCarryover(from, to BoardName, fromTiles uint32, parents []uint32) (Carryover, error) {
	if len(parents) == 0 || len(parents) > math.MaxUint32 {
		return Carryover{}, fmt.Errorf("%w: %d tiles on %s", ErrCarryover, len(parents), to)
	}
	at := make([]uint32, fromTiles+2)
	for child, parent := range parents {
		if parent > fromTiles {
			return Carryover{}, fmt.Errorf("%w: tile %d of %s comes from tile %d, past %s's %d", ErrCarryover, child+1, to,
				parent, from, fromTiles)
		}
		if parent != 0 {
			at[parent+1]++
		}
	}
	for parent := uint32(1); parent <= fromTiles; parent++ {
		at[parent+1] += at[parent]
	}
	children := make([]uint32, at[fromTiles+1])
	next := append([]uint32(nil), at...)
	for child, parent := range parents {
		if parent == 0 {
			continue
		}
		children[next[parent]] = uint32(child + 1) //nolint:gosec // bounded above.
		next[parent]++
	}
	return Carryover{from: from, to: to, parents: parents, at: at, children: children}, nil
}

func (c Carryover) From() BoardName { return c.from }
func (c Carryover) To() BoardName   { return c.to }

// Tiles is the new board's tile count.
func (c Carryover) Tiles() uint32 { return uint32(len(c.parents)) } //nolint:gosec // checked in NewCarryover.

// OldTiles is the old board's tile count.
func (c Carryover) OldTiles() uint32 { return uint32(len(c.at) - 2) } //nolint:gosec // a tile count plus two.

// Parent is the old tile a new tile comes from: 0 for none.
func (c Carryover) Parent(child uint32) uint32 {
	if child == 0 || child > c.Tiles() {
		return 0
	}
	return c.parents[child-1]
}

// Children are the new tiles an old tile's parent of, in id order: none for a coast the new board draws as sea.
func (c Carryover) Children(parent uint32) []uint32 {
	if parent == 0 || parent > c.OldTiles() {
		return nil
	}
	return c.children[c.at[parent]:c.at[parent+1]]
}

// Share is a child's part of its parent's shields: the total kept across the children, split evenly in id order, the
// remainder to the first children (10 over four: 3, 3, 2, 2). So a switch creates no protection, and loses only the
// shields of an old tile with no child.
func (c Carryover) Share(child uint32, shields int) int {
	siblings := c.Children(c.Parent(child))
	if len(siblings) == 0 || shields <= 0 {
		return 0
	}
	rank := 0
	for rank < len(siblings) && siblings[rank] != child {
		rank++
	}
	share := shields / len(siblings)
	if rank < shields%len(siblings) {
		share++
	}
	return share
}

// Heir is the child nearest its parent, where a mine of the parent goes: the first in id order of two as near. false
// for a parent with no child.
func (c Carryover) Heir(parent uint32, old Positions, new Positions) (uint32, bool) {
	from, ok := old.Position(parent)
	if !ok {
		return 0, false
	}
	heir, least := uint32(0), math.Inf(1)
	for _, child := range c.Children(parent) {
		at, ok := new.Position(child)
		if !ok {
			continue
		}
		dx, dy, dz := at.X-from.X, at.Y-from.Y, at.Z-from.Z
		if d := dx*dx + dy*dy + dz*dz; d < least {
			heir, least = child, d
		}
	}
	return heir, heir != 0
}

// Positions is where each tile of a board is.
type Positions interface {
	Position(id uint32) (Vec3, bool)
}

// Carried is what a switch of board carries: the counts its dry run answers.
type Carried struct {
	From, To BoardName
	// The new board's tiles: Carried come from an old tile, Blank from none.
	Tiles, Carried, Blank uint32
	// Tiles held by a flag on the old board, and on the new.
	HeldBefore, HeldAfter uint32
	// Old tiles with no child, and how many of them a flag held.
	Dropped, DroppedHeld        uint32
	ShieldsBefore, ShieldsAfter uint32
	MinesMoved, MinesLost       uint32
	VirusesEnded                uint32
	LocksDropped                uint32
}
