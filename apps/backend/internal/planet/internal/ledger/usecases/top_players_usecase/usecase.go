package top_players_usecase

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

type Refusals interface {
	Refusal(ctx context.Context, payer clicks.Payer) (clicks.Refusal, bool, error)
}

type In struct {
	Limit int
}

type Out struct {
	Players []ledger.Player
	Total   int
}

func New(ledger Ledger, owners Owners, refusals Refusals) *UseCase {
	return &UseCase{ledger: ledger, owners: owners, refusals: refusals}
}

type UseCase struct {
	ledger   Ledger
	owners   Owners
	refusals Refusals
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	tally := ledger.NewTally(func(ledger.Taking) bool { return true })
	u.ledger.Replay(tally.See)

	players := ledger.ByTakes(tally.Players(u.owners))
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
