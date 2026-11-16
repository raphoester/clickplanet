package lay_mine_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var ErrNoMine = errors.New("no mine to lay")

const returnTimeout = 5 * time.Second

type Gate interface {
	Lay(ctx context.Context, payer clicks.Payer, tile uint32, country string) error
}

type Hands interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
	Return(ctx context.Context, holder charges.Holder, back func(charges.Held) charges.Held) error
}

type Minefield interface {
	Lay(ctx context.Context, mine clicks.Mine) (clicks.MapVersion, error)
}

// Board tells the streams of the mine's flag, and no other.
type Board interface {
	PublishMineLaid(laid notices.MineLaid)
}

type In struct {
	TileID    uint32
	CountryID string
}

func New(gate Gate, hands Hands, minefield Minefield, board Board) *UseCase {
	return &UseCase{gate: gate, hands: hands, minefield: minefield, board: board}
}

type UseCase struct {
	gate      Gate
	hands     Hands
	minefield Minefield
	board     Board
}

// Execute spends a mine once the gate let it through, gives it back when the map refuses the tile after all, and tells
// the flag's streams of the mine laid.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	payer := clicks.PayerOf(ctx)
	if err := u.gate.Lay(ctx, payer, in.TileID, in.CountryID); err != nil {
		return err //nolint:wrapcheck // the gate names why.
	}

	holder := charges.HolderOf(payer)
	if err := u.hands.Spend(ctx, holder, charges.Held.AfterMine); err != nil {
		if errors.Is(err, charges.ErrNotHeld) {
			return ErrNoMine
		}
		return err //nolint:wrapcheck // the wallet names it.
	}

	var version clicks.MapVersion
	account, err := uuid.Parse(payer.Account)
	if err == nil {
		mine := clicks.NewMine(cpsession.AccountID(account), payer.Scope, in.CountryID, in.TileID)
		version, err = u.minefield.Lay(ctx, mine)
	}
	if err != nil {
		returning, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnTimeout)
		defer cancel()
		return errors.Join(fmt.Errorf("failed to lay the mine: %w", err),
			u.hands.Return(returning, holder, charges.Held.WithMineBack))
	}

	u.board.PublishMineLaid(notices.MineLaid{Flag: in.CountryID, Tile: in.TileID, Version: version})

	return nil
}
