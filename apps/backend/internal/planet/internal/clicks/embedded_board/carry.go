package embedded_board

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

// ErrNoCarryover is a switch between two boards this process carries no carry-over between.
var ErrNoCarryover = errors.New("this process carries no carry-over between these boards")

const (
	carryMagic   = "CPCA"
	carryVersion = 1
	carryHeader  = 24
)

// Carryover is the carry-over from one board to another (/map/carries, map/README.md#the-carry-over), read the first
// time it is asked for and kept, about 9 MB from earth-1 to earth-2: the switch reads it, and the history across it. Refused when this process carries no such file, or a file whose boards' positions
// or tile counts are not the boards' it carries: a board regenerated since is another board.
func (r *Registry) Carryover(from, to clicks.BoardName) (clicks.Carryover, error) {
	r.carried.Lock()
	defer r.carried.Unlock()
	if kept, ok := r.carryovers[[2]clicks.BoardName{from, to}]; ok {
		return kept, nil
	}
	carry, err := r.readCarryover(from, to)
	if err != nil {
		return clicks.Carryover{}, err
	}
	if r.carryovers == nil {
		r.carryovers = map[[2]clicks.BoardName]clicks.Carryover{}
	}
	r.carryovers[[2]clicks.BoardName{from, to}] = carry
	return carry, nil
}

func (r *Registry) readCarryover(from, to clicks.BoardName) (clicks.Carryover, error) {
	old, ok := r.board(from)
	if !ok {
		return clicks.Carryover{}, fmt.Errorf("%w: %s to %s", ErrNoCarryover, from, to)
	}
	fresh, ok := r.board(to)
	if !ok {
		return clicks.Carryover{}, fmt.Errorf("%w: %s to %s", ErrNoCarryover, from, to)
	}
	names, err := fs.Glob(r.carries, fmt.Sprintf("carry-%s-%s-*.bin", from, to))
	if err != nil || len(names) != 1 {
		return clicks.Carryover{}, fmt.Errorf("%w: %s to %s, %d files", ErrNoCarryover, from, to, len(names))
	}
	blob, err := fs.ReadFile(r.carries, names[0])
	if err != nil {
		return clicks.Carryover{}, fmt.Errorf("failed to read %s: %w", names[0], err)
	}
	parents, err := decodeCarry(blob, old.Manifest, fresh.Manifest)
	if err != nil {
		return clicks.Carryover{}, fmt.Errorf("%s: %w", names[0], err)
	}
	return clicks.NewCarryover(from, to, old.Manifest.Tiles, parents) //nolint:wrapcheck // it names the boards.
}

func (r *Registry) board(name clicks.BoardName) (cpboards.Board, bool) {
	for _, held := range r.boards {
		if held.board.Manifest.Name == string(name) {
			return held.board, true
		}
	}
	return cpboards.Board{}, false
}

// "CPCA" | uint32 version | uint32 from's positionsHash | uint32 to's | uint32 from's tiles | uint32 to's | a varint a
// tile of to: its parent less the one before, zigzagged.
func decodeCarry(blob []byte, from, to cpboards.Manifest) ([]uint32, error) {
	if len(blob) < carryHeader {
		return nil, fmt.Errorf("%w: %d bytes", clicks.ErrCarryover, len(blob))
	}
	if magic := string(blob[:4]); magic != carryMagic {
		return nil, fmt.Errorf("%w: magic %q, not %q", clicks.ErrCarryover, magic, carryMagic)
	}
	if version := binary.LittleEndian.Uint32(blob[4:]); version != carryVersion {
		return nil, fmt.Errorf("%w: version %d, not %d", clicks.ErrCarryover, version, carryVersion)
	}
	fromHash, toHash := binary.LittleEndian.Uint32(blob[8:]), binary.LittleEndian.Uint32(blob[12:])
	fromTiles, toTiles := binary.LittleEndian.Uint32(blob[16:]), binary.LittleEndian.Uint32(blob[20:])
	if fromHash != from.PositionsHash || fromTiles != from.Tiles {
		return nil, fmt.Errorf("%w: it comes from %d tiles hashed %08x, %s is %d hashed %08x", clicks.ErrCarryover,
			fromTiles, fromHash, from.Name, from.Tiles, from.PositionsHash)
	}
	if toHash != to.PositionsHash || toTiles != to.Tiles {
		return nil, fmt.Errorf("%w: it goes to %d tiles hashed %08x, %s is %d hashed %08x", clicks.ErrCarryover,
			toTiles, toHash, to.Name, to.Tiles, to.PositionsHash)
	}

	parents := make([]uint32, toTiles)
	rest := blob[carryHeader:]
	before := int64(0)
	for tile := range parents {
		zigzag, read := binary.Uvarint(rest)
		if read <= 0 {
			return nil, fmt.Errorf("%w: it ends at tile %d of %d", clicks.ErrCarryover, tile+1, toTiles)
		}
		rest = rest[read:]
		parent := before + int64(zigzag>>1)
		if zigzag&1 == 1 {
			parent = before - int64(zigzag>>1) - 1 //nolint:gosec // a tile id's difference.
		}
		if parent < 0 || parent > int64(fromTiles) {
			return nil, fmt.Errorf("%w: tile %d comes from tile %d, past %d", clicks.ErrCarryover, tile+1, parent, fromTiles)
		}
		parents[tile] = uint32(parent)
		before = parent
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("%w: %d bytes after the last tile", clicks.ErrCarryover, len(rest))
	}
	return parents, nil
}
