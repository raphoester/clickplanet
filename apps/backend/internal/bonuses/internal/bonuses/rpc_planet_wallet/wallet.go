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
	GetRoom(context.Context, *connect.Request[planetv1.GetRoomRequest]) (*connect.Response[planetv1.GetRoomResponse], error)
	GetChargesOf(context.Context, *connect.Request[planetv1.GetChargesOfRequest],
	) (*connect.Response[planetv1.GetChargesOfResponse], error)
}

// Lasting is how long planet knows a payment's id: an offer's for ten minutes, in memory as the offer itself is; the
// sign-in refill's for good, so a PlayerCreated told again any day pays nothing.
type Lasting bool

const (
	ForAWhile Lasting = false
	ForGood   Lasting = true
)

func New(planet Planet, lasting Lasting) Wallet {
	return Wallet{planet: planet, lasting: lasting}
}

// Wallet answers the offers' ports with planet's internal calls: the room each hand has, a payment, a hand.
type Wallet struct {
	planet  Planet
	lasting Lasting
}

func (w Wallet) Full(ctx context.Context, accounts []bonuses.Account) (map[bonuses.Account][]bonuses.Kind, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, string(account))
	}

	res, err := w.planet.GetRoom(ctx, connect.NewRequest(&planetv1.GetRoomRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet the room of %d accounts: %w", len(accounts), err)
	}

	full := make(map[bonuses.Account][]bonuses.Kind, len(res.Msg.GetRooms()))
	for _, room := range res.Msg.GetRooms() {
		for _, theirs := range room.GetFull() {
			if kind, ok := planetwords.KindOf(theirs); ok {
				full[bonuses.Account(room.GetAccountId())] = append(full[bonuses.Account(room.GetAccountId())], kind)
			}
		}
	}
	return full, nil
}

// Pay grants a caller with no account nothing, as planet's wallet holds nothing for one.
func (w Wallet) Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error) {
	if payment.Account == bonuses.NoAccount {
		return bonuses.Paid{}, nil
	}

	res, err := w.planet.GrantCharges(ctx, connect.NewRequest(&planetv1.InternalGrantChargesRequest{
		GrantId:   payment.Grant.String(),
		AccountId: string(payment.Account),
		Kind:      planetwords.Kind(payment.Kind),
		Amount:    uint32(payment.Amount), //nolint:gosec // an offer's amount, never negative.
		Lasting:   bool(w.lasting),
	}))
	if err != nil {
		return bonuses.Paid{}, fmt.Errorf("planet did not pay %s %d %s: %w", payment.Account, payment.Amount, payment.Kind, err)
	}

	return bonuses.Paid{Kept: int(res.Msg.GetKept()), Hand: planetwords.HandOf(res.Msg.GetCharges())}, nil
}

func (w Wallet) Hand(ctx context.Context, account bonuses.Account) (bonuses.Hand, error) {
	if account == bonuses.NoAccount {
		return bonuses.Hand{}, nil
	}

	res, err := w.planet.GetChargesOf(ctx, connect.NewRequest(&planetv1.GetChargesOfRequest{AccountId: string(account)}))
	if err != nil {
		return bonuses.Hand{}, fmt.Errorf("failed to ask planet what %s holds: %w", account, err)
	}
	return planetwords.HandOf(res.Msg.GetCharges()), nil
}
