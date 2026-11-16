package pay_right_answer_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// Room is the wallet's word on the kinds a hand has no room for: it is planet's.
type Room interface {
	Full(ctx context.Context, accounts []bonuses.Account) (map[bonuses.Account][]bonuses.Kind, error)
}

// Wallet pays each grant once: it is planet's, which keeps what fits.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) error
}

// Unlocks is what progression unlocked for an account: none for a guest.
type Unlocks interface {
	Unlocked(ctx context.Context, account bonuses.Account) (bonuses.Unlocked, error)
}

func New(room Room, wallet Wallet, unlocks Unlocks, rules func() bonuses.Rules) *UseCase {
	return &UseCase{room: room, wallet: wallet, unlocks: unlocks, rules: rules}
}

// UseCase pays a right answer the quizzes told. Told again, it draws again and is paid nothing more: the grant is the
// quiz's token.
type UseCase struct {
	room    Room
	wallet  Wallet
	unlocks Unlocks
	rules   func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, answer bonuses.Answer) error {
	full, err := u.room.Full(ctx, []bonuses.Account{answer.Account})
	if err != nil {
		return err //nolint:wrapcheck // the wallet named it.
	}
	unlocked, err := u.unlocks.Unlocked(ctx, answer.Account)
	if err != nil {
		return err //nolint:wrapcheck // the store named it.
	}

	reward, ok := answer.Reward(u.rules(), full[answer.Account], unlocked)
	if !ok {
		return nil
	}

	err = u.wallet.Pay(ctx, bonuses.Payment{
		Grant: answer.Grant(), Account: answer.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	return err //nolint:wrapcheck // the wallet named it.
}
