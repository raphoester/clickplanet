package pay_caught_box_usecase

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

// Out says whether the box was worth anything: the boxes, which know no kind, offer one to a hand with room for any of
// planet's kinds, and this one may have room for none the rules weigh.
type Out struct {
	Paid bool
}

func New(room Room, wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{room: room, wallet: wallet, rules: rules}
}

// UseCase draws what a box the boxes told caught holds, and pays it. Told again, it draws again and is paid nothing
// more: the grant is the box's token.
type UseCase struct {
	room   Room
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, box bonuses.CaughtBox) (Out, error) {
	full, err := u.room.Full(ctx, []bonuses.Account{box.Account})
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the wallet named it.
	}

	reward, ok := box.Reward(u.rules(), full[box.Account])
	if !ok {
		return Out{Paid: false}, nil
	}

	if err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant: box.Grant(), Account: box.Account, Kind: reward.Kind, Amount: reward.Amount,
	}); err != nil {
		return Out{}, err //nolint:wrapcheck // the wallet named it.
	}
	return Out{Paid: true}, nil
}
