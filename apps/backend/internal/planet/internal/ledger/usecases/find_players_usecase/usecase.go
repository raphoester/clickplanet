// Package find_players_usecase answers "who is painting this flag over there": the callers whose paint still holds, latest first.
package find_players_usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

const defaultLimit = 20

type Ledger interface {
	Painted(country string) []ledger.Taking
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

type Player struct {
	Scope string
	// Tiles is how many of the flag's tiles in the area this scope took and still wears.
	Tiles   int
	FirstAt time.Time
	LastAt  time.Time

	Banned      bool
	BannedUntil time.Time
	Offence     int
}

type Out struct {
	Players []Player
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

	byScope := make(map[string]*Player)
	for _, taking := range u.ledger.Painted(in.Flag) {
		if in.Area != "" && u.borders.CountryOf(taking.Tile) != in.Area {
			continue
		}
		if owner, _ := u.owners.Owner(taking.Tile); owner != in.Flag {
			continue
		}

		player, ok := byScope[taking.Scope]
		if !ok {
			player = &Player{Scope: taking.Scope, FirstAt: taking.At, LastAt: taking.At}
			byScope[taking.Scope] = player
		}
		player.Tiles++
		if taking.At.Before(player.FirstAt) {
			player.FirstAt = taking.At
		}
		if taking.At.After(player.LastAt) {
			player.LastAt = taking.At
		}
	}

	players := make([]Player, 0, len(byScope))
	for _, player := range byScope {
		players = append(players, *player)
	}

	sort.Slice(players, func(i, j int) bool {
		if !players[i].LastAt.Equal(players[j].LastAt) {
			return players[i].LastAt.After(players[j].LastAt)
		}
		return players[i].Scope < players[j].Scope
	})

	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	out := Out{Total: len(players), Players: players[:min(limit, len(players))]}
	for i := range out.Players {
		u.sentence(&out.Players[i])
	}

	return out, nil
}

func (u *UseCase) sentence(player *Player) {
	if u.bans == nil {
		return
	}

	if sentence, running := u.bans.Sentence(player.Scope); running {
		player.Banned = true
		player.BannedUntil = sentence.Until
		player.Offence = sentence.Offence
	}
}
