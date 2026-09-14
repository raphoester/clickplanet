// Package find_players_usecase answers "who is painting this flag over there": every caller who took such a tile, latest first.
package find_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(ledger.Taking)) ledger.Position
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Borders interface {
	CountryOf(tile uint32) string
}

// Bans is nil when the antibot is off.
type Bans interface {
	Sentence(scope string) (antibot.Sentence, bool)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Flag string
	// Area is the country whose ground the tiles sit on; empty is the whole map.
	Area  string
	Limit int
}

type Out struct {
	Players []ledger.Player
	// Total is how many scopes matched, before the limit.
	Total int
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

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
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
		if sentence, running := u.bans.Sentence(out.Players[i].Scope); running {
			out.Players[i].Serving(sentence)
		}
	}

	return out, nil
}
