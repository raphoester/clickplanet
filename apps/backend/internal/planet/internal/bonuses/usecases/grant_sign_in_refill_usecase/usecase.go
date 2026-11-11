package grant_sign_in_refill_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// Wallet pays the refill: it is planet's, and pays each grant once, so the refill's id makes it once per account.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error)
}

func New(wallet Wallet, rules func() bonuses.Rules) *UseCase {
	return &UseCase{wallet: wallet, rules: rules}
}

type UseCase struct {
	wallet Wallet
	rules  func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, account bonuses.Account) error {
	if !u.rules().SignInRefill {
		return nil
	}

	_, err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant: bonuses.SignInGrantOf(account), Account: account, Kind: bonuses.KindRefill, Amount: 1,
	})
	return err //nolint:wrapcheck // the wallet named it.
}
