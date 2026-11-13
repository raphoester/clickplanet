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
	Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error)
}

func New(room Room, wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{room: room, wallet: wallet, rules: rules}
}

// UseCase pays a right answer the quizzes told. Told again, it draws again and is paid nothing more: the grant is the
// quiz's token.
type UseCase struct {
	room   Room
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, answer bonuses.Answer) error {
	full, err := u.room.Full(ctx, []bonuses.Account{answer.Account})
	if err != nil {
		return err //nolint:wrapcheck // the wallet named it.
	}

	reward, ok := answer.Reward(u.rules(), full[answer.Account])
	if !ok {
		return nil
	}

	_, err = u.wallet.Pay(ctx, bonuses.Payment{
		Grant: answer.Grant(), Account: answer.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	return err //nolint:wrapcheck // the wallet named it.
}
