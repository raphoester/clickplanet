package pay_level_up_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// Wallet pays each grant once, and keeps the grant for good: a level-up is told again any day.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) (int, error)
}

func New(wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{wallet: wallet, rules: rules}
}

// UseCase pays a level-up progression told, drawn as a box. Told again, it draws again and is paid nothing more: the
// grant is progression's.
type UseCase struct {
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, levelUp bonuses.LevelUp) error {
	reward, ok := levelUp.Reward(u.rules())
	if !ok {
		return nil
	}

	_, err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant: levelUp.Grant, Account: levelUp.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	return err //nolint:wrapcheck // the wallet named it.
}
