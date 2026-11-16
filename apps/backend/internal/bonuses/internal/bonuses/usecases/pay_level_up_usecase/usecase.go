package pay_level_up_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// Room is the wallet's word on the kinds a hand has no room for: it is planet's.
type Room interface {
	Full(ctx context.Context, accounts []bonuses.Account) (map[bonuses.Account][]bonuses.Kind, error)
}

// Wallet pays each grant once, and keeps the grant for good: a level-up is told again any day.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) error
}

func New(room Room, wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{room: room, wallet: wallet, rules: rules}
}

// UseCase pays a level-up progression told, drawn as a box. Told again, it draws again and is paid nothing more: the
// grant is progression's.
type UseCase struct {
	room   Room
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, levelUp bonuses.LevelUp) error {
	full, err := u.room.Full(ctx, []bonuses.Account{levelUp.Account})
	if err != nil {
		return err //nolint:wrapcheck // the wallet named it.
	}

	reward, ok := levelUp.Reward(u.rules(), full[levelUp.Account])
	if !ok {
		return nil
	}

	err = u.wallet.Pay(ctx, bonuses.Payment{
		Grant: levelUp.Grant, Account: levelUp.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	return err //nolint:wrapcheck // the wallet named it.
}
