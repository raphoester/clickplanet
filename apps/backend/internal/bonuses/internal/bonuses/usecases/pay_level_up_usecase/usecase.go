package pay_level_up_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// Wallet fills each grant once, and keeps the grant for good: a level-up is told again any day.
type Wallet interface {
	Fill(ctx context.Context, stacks bonuses.Stacks) error
}

func New(wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{wallet: wallet, rules: rules}
}

type UseCase struct {
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, levelUp bonuses.LevelUp) error {
	stacks, ok := levelUp.Stacks(u.rules())
	if !ok {
		return nil
	}

	return u.wallet.Fill(ctx, stacks) //nolint:wrapcheck // the wallet named it.
}
