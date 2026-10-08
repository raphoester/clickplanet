package top_players_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Bans interface {
	Sentence(ctx context.Context, scope, account string) (antibot.Sentence, bool, error)
}

type In struct {
	Limit int
}

type Out struct {
	Players []ledger.Player
	Total   int
}

func New(ledger Ledger, owners Owners, bans Bans) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, bans: bans}
}

type UseCase struct {
	ledger Ledger
	owners Owners
	bans   Bans
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	tally := ledger.NewTally(func(ledger.Taking) bool { return true })
	u.ledger.Replay(tally.See)

	players := ledger.ByTakes(tally.Players(u.owners))
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
