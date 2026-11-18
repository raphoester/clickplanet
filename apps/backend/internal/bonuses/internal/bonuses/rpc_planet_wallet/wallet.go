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
	FillCharges(context.Context, *connect.Request[planetv1.FillChargesRequest],
	) (*connect.Response[planetv1.FillChargesResponse], error)
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

// Fill tops each stack up to planet's size for it, and planet keeps the grant for good, lasting or not.
func (w Wallet) Fill(ctx context.Context, stacks bonuses.Stacks) error {
	if stacks.Account == bonuses.NoAccount {
		return nil
	}

	kinds := make([]planetv1.BonusKind, 0, len(stacks.Kinds))
	for _, kind := range stacks.Kinds {
		kinds = append(kinds, planetwords.Kind(kind))
	}
	if _, err := w.planet.FillCharges(ctx, connect.NewRequest(&planetv1.FillChargesRequest{
		GrantId: stacks.Grant.String(), AccountId: string(stacks.Account), Kinds: kinds,
	})); err != nil {
		return fmt.Errorf("planet did not fill %s's %v: %w", stacks.Account, stacks.Kinds, err)
	}
	return nil
}
