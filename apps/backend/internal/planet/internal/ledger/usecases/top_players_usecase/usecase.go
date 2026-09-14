// Package top_players_usecase answers "who holds the most of the map": the callers whose paint still holds on the most tiles.
package top_players_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	All() []ledger.Taking
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// Bans is nil when the antibot is off.
type Bans interface {
	Sentence(scope string) (antibot.Sentence, bool)
}

type In struct {
	Limit int
}

type Out struct {
	Players []ledger.Player
	// Total is how many scopes still wear paint, before the limit.
	Total int
}

func New(ledger Ledger, owners Owners, bans Bans) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, bans: bans}
}

type UseCase struct {
	ledger Ledger
	owners Owners
	bans   Bans
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	var worn []ledger.Taking
	for _, taking := range u.ledger.All() {
		if owner, _ := u.owners.Owner(taking.Tile); taking.WornBy(owner) {
			worn = append(worn, taking)
		}
	}

	players := ledger.ByTiles(ledger.Players(worn))
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
