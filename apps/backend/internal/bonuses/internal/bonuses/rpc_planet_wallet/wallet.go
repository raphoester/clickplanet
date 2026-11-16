package rpc_planet_wallet

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
)

type Planet interface {
	GrantCharges(context.Context, *connect.Request[planetv1.InternalGrantChargesRequest],
	) (*connect.Response[planetv1.InternalGrantChargesResponse], error)
}

// Lasting is how long planet knows a payment's id: a box's or a right answer's for ten minutes, in memory as the offer
// itself was; the sign-in refill's for good, so a PlayerCreated told again any day pays nothing.
type Lasting bool

const (
	ForAWhile Lasting = false
	ForGood   Lasting = true
)

func New(planet Planet, lasting Lasting) Wallet {
	return Wallet{planet: planet, lasting: lasting}
}

// Wallet answers the bonuses module's ports with planet's internal call: a payment, and how many of it planet kept.
type Wallet struct {
	planet  Planet
	lasting Lasting
}

// Pay grants a caller with no account nothing, as planet's wallet holds nothing for one. It answers how many of the kind
// planet kept: fewer than the amount for a full stack, none for a kind planet's rules turn off or a grant paid before.
func (w Wallet) Pay(ctx context.Context, payment bonuses.Payment) (int, error) {
	if payment.Account == bonuses.NoAccount {
		return 0, nil
	}

	res, err := w.planet.GrantCharges(ctx, connect.NewRequest(&planetv1.InternalGrantChargesRequest{
		GrantId:   payment.Grant.String(),
		AccountId: string(payment.Account),
		Kind:      planetwords.Kind(payment.Kind),
		Amount:    uint32(payment.Amount), //nolint:gosec // an offer's amount, never negative.
		Lasting:   bool(w.lasting),
	}))
	if err != nil {
		return 0, fmt.Errorf("planet did not pay %s %d %s: %w", payment.Account, payment.Amount, payment.Kind, err)
	}
	return int(res.Msg.GetKept()), nil
}
