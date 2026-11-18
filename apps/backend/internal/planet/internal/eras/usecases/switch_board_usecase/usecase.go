package switch_board_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	// ErrNotTheWorld is a switch of another map than the world's: an arena plays the board its set names.
	ErrNotTheWorld = errors.New("only the world's map switches boards")
	// ErrArenaPlays is a switch while an arena plays in the world's place.
	ErrArenaPlays = errors.New("an arena is open: the world's map switches only while it plays")
	// ErrPlaysAlready is a switch onto the board the world plays: a second call.
	ErrPlaysAlready = errors.New("the world plays that board already")
)

type Tiles interface {
	Stage() clicks.Stage
	Carried(onto clicks.Stage, carry clicks.Carryover) (clicks.Carried, error)
	Switch(ctx context.Context, onto clicks.Stage, carry clicks.Carryover,
		keep func(ctx context.Context, carried clicks.Carried) (bool, error), recorded func(carried clicks.Carried),
	) (clicks.Carried, bool, error)
}

// Boards are the boards this process carries: eras.Grounds.
type Boards interface {
	GroundOf(id clicks.BoardID) (*clicks.Ground, error)
	Named(name clicks.BoardName) (eras.Board, *clicks.Ground, error)
}

// Carryovers are the carry-overs between two boards this process carries.
type Carryovers interface {
	Carryover(from, to clicks.BoardName) (clicks.Carryover, error)
}

// Eras keeps the era the world's map starts on the new board, in the switch's transaction.
type Eras interface {
	Start(ctx context.Context, played clicks.Stage, at time.Time) error
}

type Ledger interface {
	Append(on clicks.Stage, event ledger.Event)
}

type Command struct {
	Map    clicks.MapID
	Board  clicks.BoardName
	DryRun bool
}

// Out is what the switch carried, or would carry on a dry run, and when it switched.
type Out struct {
	Carried    clicks.Carried
	SwitchedAt time.Time
}

type Executor interface {
	Execute(ctx context.Context, command Command) (Out, error)
}

func New(tiles Tiles, boards Boards, carryovers Carryovers, kept Eras, ledger Ledger, clock cptime.Clock) *UseCase {
	return &UseCase{tiles: tiles, boards: boards, carryovers: carryovers, eras: kept, ledger: ledger, clock: clock}
}

type UseCase struct {
	tiles      Tiles
	boards     Boards
	carryovers Carryovers
	eras       Eras
	ledger     Ledger
	clock      cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// Execute moves the world's map onto the board named, its land carried over, in one transaction under the map's locks
// with the new era, then records one board_switched event, which is no take: the relay tells it as BoardSwitched, and
// nobody gains tiles, XP or a mission's progress. A dry run answers the counts and changes nothing. The world's stage
// is read again under the locks, so a second call, or one made while an arena plays, is refused.
func (u *UseCase) Execute(ctx context.Context, command Command) (Out, error) {
	if command.Map != clicks.World {
		return Out{}, fmt.Errorf("%w: %q", ErrNotTheWorld, command.Map)
	}
	to, _, err := u.boards.Named(command.Board)
	if err != nil {
		return Out{}, fmt.Errorf("cannot play %q: %w", command.Board, err)
	}
	playing := u.tiles.Stage()
	if playing.Map != clicks.World {
		return Out{}, fmt.Errorf("%w: arena %s", ErrArenaPlays, playing.Map)
	}
	if playing.Board == to.ID {
		return Out{}, fmt.Errorf("%w: %s", ErrPlaysAlready, to.Name)
	}
	from, err := u.boards.GroundOf(playing.Board)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the boards name it.
	}
	carry, err := u.carryovers.Carryover(from.Name(), to.Name)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the registry names it.
	}
	onto := clicks.World.On(to.ID)

	if command.DryRun {
		carried, err := u.tiles.Carried(onto, carry)
		return Out{Carried: carried}, refusalOf(err)
	}

	at := u.clock.Now()
	carried, _, err := u.tiles.Switch(ctx, onto, carry,
		func(ctx context.Context, _ clicks.Carried) (bool, error) {
			if err := u.eras.Start(ctx, onto, at); err != nil {
				return false, err //nolint:wrapcheck // the store names it.
			}
			return true, nil
		},
		func(carried clicks.Carried) {
			u.ledger.Append(onto, ledger.Switching{
				From: carried.From, To: carried.To, FromTiles: carry.OldTiles(), ToTiles: carry.Tiles(), At: at,
			})
		})
	if err != nil {
		return Out{Carried: carried}, refusalOf(err)
	}
	return Out{Carried: carried, SwitchedAt: at}, nil
}

// The storage reads the stage again under its locks: a swap or a switch since the read above is refused as such.
func refusalOf(err error) error {
	switch {
	case errors.Is(err, clicks.ErrNotInPlay):
		return fmt.Errorf("%w: %w", ErrArenaPlays, err)
	case errors.Is(err, clicks.ErrPlaysAlready):
		return fmt.Errorf("%w: %w", ErrPlaysAlready, err)
	default:
		return err
	}
}
