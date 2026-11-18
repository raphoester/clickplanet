// Package eras keeps which board each map plays, and since when: the boards planet keeps a row of, and each map's eras.
// A tile id means a place only on its board, so every write names the board its map plays (clicks.Stage), and every
// answer that names tiles names its board.
package eras

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	// ErrNoEra is a map with no era: the world got its era from the migration that made eras, an arena gets one as it
	// opens.
	ErrNoEra = errors.New("the map has no era")
	// ErrNotCarried is a map whose era plays a board this process does not carry: its tile ids would land on other
	// ground.
	ErrNotCarried = errors.New("the map plays a board this process does not carry")
	// ErrUnkept is a board this process carries that planet keeps no row of, or keeps under another hash: another hash is
	// tiles that moved, another board under the same name.
	ErrUnkept = errors.New("this process carries a board planet does not keep")
)

// Board is a board as planet keeps it: its row, its name, and the hash of its positions in tile order (the manifest's
// positionsHash, in 8 hex), which moves exactly when a tile's id or place does.
type Board struct {
	ID   clicks.BoardID
	Name clicks.BoardName
	Hash string
}

// Era is a stretch of a map on one board, from StartedAt on: a map's latest era is the board it plays.
type Era struct {
	Map       clicks.MapID
	Board     Board
	StartedAt time.Time
}

// Stage is the map on the era's board.
func (e Era) Stage() clicks.Stage {
	return e.Map.On(e.Board.ID)
}

// Carried is a board as this process's files name it: its directory and its positions' hash.
type Carried struct {
	Name clicks.BoardName
	Hash string
}

// Boards are the boards this process carries, each with the row planet keeps of it.
type Boards struct {
	carried []Board
}

// BoardsOf matches each board carried with planet's row of it, by name and hash.
func BoardsOf(kept []Board, carried ...Carried) (Boards, error) {
	boards := Boards{carried: make([]Board, 0, len(carried))}
	for _, files := range carried {
		row, found := rowOf(kept, files.Name)
		switch {
		case !found:
			return Boards{}, fmt.Errorf("%w: no row names board %s", ErrUnkept, files.Name)
		case row.Hash != files.Hash:
			return Boards{}, fmt.Errorf("%w: board %s hashes %s here and %s in planet's boards", ErrUnkept, files.Name,
				files.Hash, row.Hash)
		}
		boards.carried = append(boards.carried, row)
	}
	return boards, nil
}

func rowOf(kept []Board, name clicks.BoardName) (Board, bool) {
	for _, board := range kept {
		if board.Name == name {
			return board, true
		}
	}
	return Board{}, false
}

// Board is the board of id, when this process carries it.
func (b Boards) Board(id clicks.BoardID) (Board, bool) {
	for _, board := range b.carried {
		if board.ID == id {
			return board, true
		}
	}
	return Board{}, false
}

// Name is the name of the board of id, empty when this process does not carry it.
func (b Boards) Name(id clicks.BoardID) clicks.BoardName {
	board, _ := b.Board(id)
	return board.Name
}

// Stage is the stage of a map's era, when this process carries its board: the boot refuses any other.
func (b Boards) Stage(era Era) (clicks.Stage, error) {
	if _, carried := b.Board(era.Board.ID); !carried {
		return clicks.Stage{}, fmt.Errorf("%w: map %s plays board %s (row %d) since %s", ErrNotCarried, era.Map,
			era.Board.Name, era.Board.ID, era.StartedAt.UTC().Format(time.RFC3339))
	}
	return era.Stage(), nil
}
