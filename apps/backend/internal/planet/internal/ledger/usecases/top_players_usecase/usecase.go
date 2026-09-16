// Package top_players_usecase answers "who paints the most of the map": the callers with the most takes, every flag.
package top_players_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(ledger.Taking)) ledger.Position
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// Bans is nil when the antibot is off.
type Bans interface {
	Sentence(scope, account string) (antibot.Sentence, bool)
}

type In struct {
	Limit int
}

type Out struct {
	Players []ledger.Player
	// Total is how many players took a tile, before the limit.
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
	tally := ledger.NewTally(func(ledger.Taking) bool { return true })
	u.ledger.Replay(tally.See)

	players := ledger.ByTakes(tally.Players(u.owners))
	out := Out{Total: len(players), Players: ledger.Top(players, in.Limit)}

	if u.bans == nil {
		return out, nil
	}

	for i := range out.Players {
		if sentence, running := u.bans.Sentence(out.Players[i].Scope, out.Players[i].Account); running {
			out.Players[i].Serving(sentence)
		}
	}

	return out, nil
}
