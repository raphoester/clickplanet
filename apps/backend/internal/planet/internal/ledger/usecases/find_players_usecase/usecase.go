package find_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(on clicks.Stage, event ledger.Event)) ledger.Position
}

// Owners is the map that plays, on its board: only that board's events are counted, as each names its board.
type Owners interface {
	Stage() clicks.Stage
	Owner(tile uint32) (string, bool)
}

type Borders interface {
	CountryOf(tile uint32) string
}

type Refusals interface {
	Refusals(ctx context.Context, payers []clicks.Payer) (map[clicks.Payer]clicks.Refusal, error)
}

// A flag closed is known: its players are still found.
type Flags interface {
	Known(ctx context.Context, flag string) (bool, error)
}

type In struct {
	Flag  string
	Area  string
	Limit int
}

type Out struct {
	Players []ledger.Player
	Total   int
}

func New(ledger Ledger, owners Owners, borders Borders, refusals Refusals, countries Flags) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, borders: borders, refusals: refusals, countries: countries}
}

type UseCase struct {
	ledger    Ledger
	owners    Owners
	borders   Borders
	refusals  Refusals
	countries Flags
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if err := u.check(ctx, "flag", in.Flag); err != nil {
		return Out{}, err
	}
	if in.Area != "" {
		if err := u.check(ctx, "area", in.Area); err != nil {
			return Out{}, err
		}
	}

	tally := ledger.NewTally(func(taking ledger.Taking) bool {
		return taking.Country == in.Flag && (in.Area == "" || u.borders.CountryOf(taking.Tile) == in.Area)
	})
	u.ledger.Replay(ledger.OnBoard(u.owners.Stage().Board, tally.See))

	players := tally.Players(u.owners)
	out := Out{Total: len(players), Players: ledger.Top(players, in.Limit)}

	payers := make([]clicks.Payer, len(out.Players))
	for i, player := range out.Players {
		payers[i] = clicks.Payer{Scope: player.Scope, Account: player.Account}
	}
	refused, err := u.refusals.Refusals(ctx, payers)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the refusals: %w", err)
	}
	for i, payer := range payers {
		if refusal, ok := refused[payer]; ok {
			out.Players[i].Refused(refusal)
		}
	}

	return out, nil
}

func (u *UseCase) check(ctx context.Context, what string, flag string) error {
	known, err := u.countries.Known(ctx, flag)
	if err != nil {
		return fmt.Errorf("failed to read whether the %s %q is a flag: %w", what, flag, err)
	}
	if !known {
		return fmt.Errorf("%w: %s %q", clicks.ErrUnknownCountry, what, flag)
	}
	return nil
}
