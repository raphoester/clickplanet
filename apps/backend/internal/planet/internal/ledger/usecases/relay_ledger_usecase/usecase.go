package relay_ledger_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	ReplaySaved(from ledger.Position, most int, see func(ledger.Position, clicks.Stage, ledger.Event)) ledger.Position
}

type Cursor interface {
	Relayed(ctx context.Context) (ledger.Position, error)
	SaveRelayed(ctx context.Context, position ledger.Position) error
}

type Borders interface {
	CountryOf(tile uint32) string
	LandmassOf(tile uint32) clicks.LandmassID
}

type Positions interface {
	Position(tile uint32) (clicks.Vec3, bool)
}

// Boards are the name, the borders and the positions of each board this process carries: an event's tiles are ids of
// its board.
type Boards interface {
	Of(board clicks.BoardID) (clicks.BoardName, Borders, Positions, bool)
}

// OnGrounds reads each board's from its ground.
type OnGrounds struct {
	Grounds clicks.Grounds
}

func (g OnGrounds) Of(board clicks.BoardID) (clicks.BoardName, Borders, Positions, bool) {
	ground, err := g.Grounds.GroundOf(board)
	if err != nil {
		return "", nil, nil, false
	}
	return ground.Name(), ground.Borders(), ground.Geography(), true
}

// An event on a board this process does not carry tells its tiles with no ground and no point.
type noBoard struct{}

func (noBoard) CountryOf(uint32) string             { return "" }
func (noBoard) LandmassOf(uint32) clicks.LandmassID { return 0 }
func (noBoard) Position(uint32) (clicks.Vec3, bool) { return clicks.Vec3{}, false }

type Sender interface {
	Send(ctx context.Context, events ...proto.Message) error
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const Batch = 1000

type UseCase struct {
	ledger Ledger
	cursor Cursor
	boards Boards
	events Sender
}

var _ Executor = (*UseCase)(nil)

func New(ledger Ledger, cursor Cursor, boards Boards, events Sender) *UseCase {
	return &UseCase{ledger: ledger, cursor: cursor, boards: boards, events: events}
}

// A crash between the send and the save sends the batch again: each fact names its position.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	from, err := u.cursor.Relayed(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read how far the ledger was told: %w", err)
	}

	var facts []proto.Message
	next := u.ledger.ReplaySaved(from, Batch, func(position ledger.Position, on clicks.Stage, event ledger.Event) {
		facts = append(facts, u.factsOf(position, on, event)...)
	})
	if next == from {
		return 0, nil
	}

	if err := u.events.Send(ctx, facts...); err != nil {
		return 0, fmt.Errorf("failed to tell %d facts of the ledger: %w", len(facts), err)
	}
	if err := u.cursor.SaveRelayed(ctx, next); err != nil {
		return 0, fmt.Errorf("failed to save how far the ledger was told: %w", err)
	}
	return int(next - from), nil //nolint:gosec // at most Batch.
}

// The ground and the point of each tile are its board's, the board the event names, which the facts name too.
func (u *UseCase) factsOf(position ledger.Position, on clicks.Stage, event ledger.Event) []proto.Message {
	var (
		board     clicks.BoardName
		borders   Borders   = noBoard{}
		positions Positions = noBoard{}
	)
	if name, carried, at, ok := u.boards.Of(on.Board); ok {
		board, borders, positions = name, carried, at
	}
	facts := takenOf(position, event, board, borders, positions)
	facts = append(facts, struckOf(position, event)...)

	switch told := event.(type) {
	case ledger.Bombing:
		facts = append(facts, landedOf(position, told, borders))
	case ledger.Fortifying:
		facts = append(facts, fortifiedOf(position, told, board))
	case ledger.Shielding:
		facts = append(facts, shieldPlacedOf(position, told))
	case ledger.KnockingOut:
		facts = append(facts, knockedOutOf(position, told))
	case ledger.Switching:
		facts = append(facts, switchedOf(position, on, told))
	}
	return facts
}

// PartTiles is the most tiles one TilesTaken or ShieldsStruck holds: an enclose of a continent is a few entries of
// some 200 KB each on the stream, never one of megabytes.
const PartTiles = 4096

// The takes of one event, in parts: a tile's index counts the takes told before it in the event, across the parts.
func takenOf(position ledger.Position, event ledger.Event, board clicks.BoardName, borders Borders, positions Positions) []proto.Message {
	var parts []proto.Message
	var part *planetv1.TilesTaken
	var index uint32
	event.Replay(func(taking ledger.Taking) {
		if taking.Account == "" || taking.Cleared() {
			return
		}
		if part == nil || len(part.Tiles) == PartTiles {
			part = &planetv1.TilesTaken{
				AccountId: taking.Account,
				Country:   taking.Country,
				TakenAt:   timestamppb.New(taking.At),
				Position:  uint64(position),
				Board:     string(board),
			}
			parts = append(parts, part)
		}
		tile := &planetv1.TakenTile{
			TileId:   taking.Tile,
			Index:    index,
			Previous: taking.Previous,
			Ground:   borders.CountryOf(taking.Tile),
		}
		if at, ok := positions.Position(taking.Tile); ok {
			tile.Point = &planetv1.GlobePoint{X: at.X, Y: at.Y, Z: at.Z}
		}
		part.Tiles = append(part.Tiles, tile)
		index++
	})
	return parts
}

func struckOf(position ledger.Position, event ledger.Event) []proto.Message {
	var parts []proto.Message
	var part *planetv1.ShieldsStruck
	var index uint32
	event.Strikes(func(strike ledger.Strike) {
		if part == nil || len(part.Tiles) == PartTiles {
			part = &planetv1.ShieldsStruck{
				AccountId: strike.Account,
				Country:   strike.Country,
				StruckAt:  timestamppb.New(strike.At),
				Position:  uint64(position),
			}
			parts = append(parts, part)
		}
		part.Tiles = append(part.Tiles, &planetv1.StruckTile{TileId: strike.Tile, Index: index, Owner: strike.Owner})
		index++
	})
	return parts
}

func landedOf(position ledger.Position, bombing ledger.Bombing, borders Borders) *planetv1.BombLanded {
	landed := &planetv1.BombLanded{
		Country:   bombing.Blast.CountryID,
		TileId:    bombing.Blast.Tile,
		Cleared:   uint32(len(bombing.Blast.Cleared)), //nolint:gosec // bounded by the map.
		LandedAt:  timestamppb.New(bombing.At),
		AccountId: bombing.Account,
		Position:  uint64(position),
	}
	if bombing.Blast.Tile != 0 {
		landed.Ground = borders.CountryOf(bombing.Blast.Tile)
		landed.LandmassId = uint32(borders.LandmassOf(bombing.Blast.Tile))
	}
	return landed
}

func fortifiedOf(position ledger.Position, fortifying ledger.Fortifying, board clicks.BoardName) *planetv1.Fortified {
	return &planetv1.Fortified{
		Country:     fortifying.Country,
		Ground:      fortifying.Ground,
		LandmassId:  uint32(fortifying.Landmass),
		Tiles:       uint32(fortifying.Tiles), //nolint:gosec // a landmass's tiles, bounded by the map.
		FortifiedAt: timestamppb.New(fortifying.At),
		AccountId:   fortifying.Account,
		Position:    uint64(position),
		Board:       string(board),
	}
}

func shieldPlacedOf(position ledger.Position, shielding ledger.Shielding) *planetv1.ShieldPlaced {
	return &planetv1.ShieldPlaced{
		AccountId: shielding.Account,
		TileId:    shielding.Tile,
		Country:   shielding.Country,
		Shields:   uint32(shielding.Shields), //nolint:gosec // a tile's shields, bounded by charges.shield.perTile.
		PlacedAt:  timestamppb.New(shielding.At),
		Position:  uint64(position),
	}
}

func knockedOutOf(position ledger.Position, knockingOut ledger.KnockingOut) *planetv1.FlagKnockedOut {
	return &planetv1.FlagKnockedOut{
		Country:      knockingOut.Out,
		By:           knockingOut.Country,
		TileId:       knockingOut.Tile,
		AccountId:    knockingOut.Account,
		Cleared:      uint32(len(knockingOut.Cleared)), //nolint:gosec // bounded by the map.
		KnockedOutAt: timestamppb.New(knockingOut.At),
		Position:     uint64(position),
	}
}

func switchedOf(position ledger.Position, on clicks.Stage, switching ledger.Switching) *planetv1.BoardSwitched {
	return &planetv1.BoardSwitched{
		Map:        string(on.Map),
		From:       string(switching.From),
		To:         string(switching.To),
		FromTiles:  switching.FromTiles,
		ToTiles:    switching.ToTiles,
		SwitchedAt: timestamppb.New(switching.At),
		Position:   uint64(position),
	}
}
