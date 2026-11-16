package pay_caught_box_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

// Wallet pays each grant once: it is planet's, which keeps what fits, and answers how many of the kind it kept.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) (int, error)
}

// Unlocks is what progression unlocked for an account: none for a guest.
type Unlocks interface {
	Unlocked(ctx context.Context, account bonuses.Account) (bonuses.Unlocked, error)
}

// Out says whether the box was worth anything: the draw ignores the hand, so a kind drawn into a full stack, or one
// planet's rules turn off, is kept none of, and a box whose kinds the account may be drawn all weigh 0 draws nothing.
type Out struct {
	Paid bool
}

func New(wallet Wallet, unlocks Unlocks, rules func() bonuses.Rules) *UseCase {
	return &UseCase{wallet: wallet, unlocks: unlocks, rules: rules}
}

// UseCase draws what a box the boxes told caught holds, and pays it. Told again, it draws again and is paid nothing
// more: the grant is the box's token.
type UseCase struct {
	wallet  Wallet
	unlocks Unlocks
	rules   func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, box bonuses.CaughtBox) (Out, error) {
	unlocked, err := u.unlocks.Unlocked(ctx, box.Account)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the store named it.
	}

	reward, ok := box.Reward(u.rules(), unlocked)
	if !ok {
		return Out{Paid: false}, nil
	}

	kept, err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant: box.Grant(), Account: box.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the wallet named it.
	}
	return Out{Paid: kept > 0}, nil
}
