package find_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
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

type Bans interface {
	Sentence(ctx context.Context, scope, account string) (antibot.Sentence, bool, error)
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

func New(ledger Ledger, owners Owners, borders Borders, bans Bans, countries CountryChecker) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, borders: borders, bans: bans, countries: countries}
}

type UseCase struct {
	ledger    Ledger
	owners    Owners
	borders   Borders
	bans      Bans
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

	if u.bans == nil {
		return out, nil
	}

	for i := range out.Players {
		sentence, running, err := u.bans.Sentence(ctx, out.Players[i].Scope, out.Players[i].Account)
		if err != nil {
			return Out{}, fmt.Errorf("failed to read the bans: %w", err)
		}
		if running {
			out.Players[i].Serving(sentence)
		}
	}

	return out, nil
}
