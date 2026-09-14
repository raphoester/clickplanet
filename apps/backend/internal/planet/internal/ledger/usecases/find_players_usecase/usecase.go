// Package find_players_usecase answers "who is painting this flag over there": the callers whose paint still holds, latest first.
package find_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	PaintedWith(country string) []ledger.Taking
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

	var worn []ledger.Taking
	for _, taking := range u.ledger.PaintedWith(in.Flag) {
		if in.Area != "" && u.borders.CountryOf(taking.Tile) != in.Area {
			continue
		}
		if owner, _ := u.owners.Owner(taking.Tile); taking.WornBy(owner) {
			worn = append(worn, taking)
		}
	}

	players := ledger.Players(worn)
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
