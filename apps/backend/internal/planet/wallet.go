package planet

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
)

// wallet answers the ports the offers declare (bonuses.Room, and each use case's Wallet) from planet's charges. It is
// the seam between them: the offers name a kind and an account their own way, and the wallet keeps its sizes to itself.
type wallet struct {
	hands charges.Charges
	rules func() charges.Rules
	pay   *pay_usecase.UseCase
}

func newWallet(hands charges.Charges, rules func() charges.Rules, pay *pay_usecase.UseCase) wallet {
	return wallet{hands: hands, rules: rules, pay: pay}
}

func (w wallet) Full(ctx context.Context, accounts []bonuses.Account) (map[bonuses.Account][]bonuses.Kind, error) {
	rules := w.rules()

	full := make(map[bonuses.Account][]bonuses.Kind, len(accounts))
	for _, account := range accounts {
		held, err := w.hands.Held(ctx, charges.Holder(account))
		if err != nil {
			return nil, err
		}
		for _, kind := range held.Full(rules) {
			full[account] = append(full[account], bonuses.Kind(kind))
		}
	}

	return full, nil
}

func (w wallet) Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error) {
	out, err := w.pay.Execute(ctx, pay_usecase.In{
		Grant:  charges.GrantID(payment.Grant),
		Holder: charges.Holder(payment.Account),
		Kind:   charges.Kind(payment.Kind),
		Amount: payment.Amount,
	})
	if err != nil {
		return bonuses.Paid{}, err
	}

	return bonuses.Paid{Kept: out.Kept, Hand: bonuses.Hand(out.Held)}, nil
}

func (w wallet) Hand(ctx context.Context, account bonuses.Account) (bonuses.Hand, error) {
	held, err := w.hands.Held(ctx, charges.Holder(account))
	if err != nil {
		return bonuses.Hand{}, err
	}

	return bonuses.Hand(held), nil
}
