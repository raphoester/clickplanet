package find_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Borders interface {
	CountryOf(tile uint32) string
}

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
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

func New(ledger Ledger, owners Owners, borders Borders, refusals Refusals, countries CountryChecker) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, borders: borders, refusals: refusals, countries: countries}
}

type UseCase struct {
	ledger    Ledger
	owners    Owners
	borders   Borders
	refusals  Refusals
	countries CountryChecker
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if !u.countries.CheckCountry(in.Flag) {
		return Out{}, fmt.Errorf("%w: flag %q", clicks.ErrUnknownCountry, in.Flag)
	}
	if in.Area != "" && !u.countries.CheckCountry(in.Area) {
		return Out{}, fmt.Errorf("%w: area %q", clicks.ErrUnknownCountry, in.Area)
	}

	tally := ledger.NewTally(func(taking ledger.Taking) bool {
		return taking.Country == in.Flag && (in.Area == "" || u.borders.CountryOf(taking.Tile) == in.Area)
	})
	u.ledger.Replay(tally.See)

	players := tally.Players(u.owners)
	out := Out{Total: len(players), Players: ledger.Top(players, in.Limit)}

	for i, player := range out.Players {
		refusal, refused, err := u.refusals.Refusal(ctx, clicks.Payer{Scope: player.Scope, Account: player.Account})
		if err != nil {
			return Out{}, fmt.Errorf("failed to read the refusals: %w", err)
		}
		if refused {
			out.Players[i].Refused(refusal)
		}
	}

	return out, nil
}
