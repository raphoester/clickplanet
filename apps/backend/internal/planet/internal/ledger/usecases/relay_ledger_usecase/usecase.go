package relay_ledger_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
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
}

type Sender interface {
	Send(ctx context.Context, events ...proto.Message) error
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

const Batch = 1000

type UseCase struct {
	ledger  Ledger
	cursor  Cursor
	borders Borders
	events  Sender
}

var _ Executor = (*UseCase)(nil)

func New(ledger Ledger, cursor Cursor, borders Borders, events Sender) *UseCase {
	return &UseCase{ledger: ledger, cursor: cursor, borders: borders, events: events}
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
	var facts []proto.Message
	var index uint32
	event.Replay(func(taking ledger.Taking) {
		if taking.Account == "" || taking.Cleared() {
			return
		}
		facts = append(facts, &planetv1.TileTaken{
			AccountId: taking.Account,
			TileId:    taking.Tile,
			Country:   taking.Country,
			TakenAt:   timestamppb.New(taking.At),
			Position:  uint64(position),
			Previous:  taking.Previous,
			Ground:    u.borders.CountryOf(taking.Tile),
			Index:     index,
		})
		index++
	})

	switch told := event.(type) {
	case ledger.Bombing:
		facts = append(facts, u.landedOf(position, told))
	case ledger.Fortifying:
		facts = append(facts, fortifiedOf(position, told))
	}
	return facts
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
