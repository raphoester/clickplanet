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
	Refusals(ctx context.Context, payers []clicks.Payer) (map[clicks.Payer]clicks.Refusal, error)
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
