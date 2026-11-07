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
	ReplaySaved(from ledger.Position, most int, see func(ledger.Position, ledger.Event)) ledger.Position
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

type Sender interface {
	Send(ctx context.Context, events ...proto.Message) error
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const Batch = 1000

type UseCase struct {
	ledger    Ledger
	cursor    Cursor
	borders   Borders
	positions Positions
	events    Sender
}

var _ Executor = (*UseCase)(nil)

func New(ledger Ledger, cursor Cursor, borders Borders, positions Positions, events Sender) *UseCase {
	return &UseCase{ledger: ledger, cursor: cursor, borders: borders, positions: positions, events: events}
}

// A crash between the send and the save sends the batch again: each fact names its position.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	from, err := u.cursor.Relayed(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read how far the ledger was told: %w", err)
	}

	var facts []proto.Message
	next := u.ledger.ReplaySaved(from, Batch, func(position ledger.Position, event ledger.Event) {
		facts = append(facts, u.factsOf(position, event)...)
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

func (u *UseCase) factsOf(position ledger.Position, event ledger.Event) []proto.Message {
	facts := u.takenOf(position, event)
	facts = append(facts, struckOf(position, event)...)

	switch told := event.(type) {
	case ledger.Bombing:
		facts = append(facts, u.landedOf(position, told))
	case ledger.Fortifying:
		facts = append(facts, fortifiedOf(position, told))
	case ledger.Shielding:
		facts = append(facts, shieldPlacedOf(position, told))
	}
	return facts
}

// PartTiles is the most tiles one TilesTaken or ShieldsStruck holds: an enclose of a continent is a few entries of
// some 200 KB each on the stream, never one of megabytes.
const PartTiles = 4096

// The takes of one event, in parts: a tile's index counts the takes told before it in the event, across the parts.
func (u *UseCase) takenOf(position ledger.Position, event ledger.Event) []proto.Message {
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
			}
			parts = append(parts, part)
		}
		tile := &planetv1.TakenTile{
			TileId:   taking.Tile,
			Index:    index,
			Previous: taking.Previous,
			Ground:   u.borders.CountryOf(taking.Tile),
		}
		if at, ok := u.positions.Position(taking.Tile); ok {
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

func (u *UseCase) landedOf(position ledger.Position, bombing ledger.Bombing) *planetv1.BombLanded {
	landed := &planetv1.BombLanded{
		Country:   bombing.Blast.CountryID,
		TileId:    bombing.Blast.Tile,
		Cleared:   uint32(len(bombing.Blast.Cleared)), //nolint:gosec // bounded by the map.
		LandedAt:  timestamppb.New(bombing.At),
		AccountId: bombing.Account,
		Position:  uint64(position),
	}
	if bombing.Blast.Tile != 0 {
		landed.Ground = u.borders.CountryOf(bombing.Blast.Tile)
		landed.LandmassId = uint32(u.borders.LandmassOf(bombing.Blast.Tile))
	}
	return landed
}

func fortifiedOf(position ledger.Position, fortifying ledger.Fortifying) *planetv1.Fortified {
	return &planetv1.Fortified{
		Country:     fortifying.Country,
		Ground:      fortifying.Ground,
		LandmassId:  uint32(fortifying.Landmass),
		Tiles:       uint32(fortifying.Tiles), //nolint:gosec // a landmass's tiles, bounded by the map.
		FortifiedAt: timestamppb.New(fortifying.At),
		AccountId:   fortifying.Account,
		Position:    uint64(position),
	}
}

func shieldPlacedOf(position ledger.Position, shielding ledger.Shielding) *planetv1.ShieldPlaced {
	return &planetv1.ShieldPlaced{
		AccountId: shielding.Account,
		TileId:    shielding.Tile,
		Country:   shielding.Country,
		Shields:   uint32(shielding.Shields), //nolint:gosec // a tile's shields, bounded by bonuses.shield.perTile.
		PlacedAt:  timestamppb.New(shielding.At),
		Position:  uint64(position),
	}
}
