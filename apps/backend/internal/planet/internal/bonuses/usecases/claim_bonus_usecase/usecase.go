package claim_bonus_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// One error for every refusal: telling them apart helps a script guessing tokens.
var ErrNoSuchBonus = errors.New("no bonus to claim")

const grantTimeout = 5 * time.Second

type Registry interface {
	Claim(token string, entrant bonuses.Entrant, scope string) (bonuses.Reward, bool)
	Publish(taken bonuses.Taken)
}

// Wallet pays what a box grants: it is planet's, which keeps what fits.
type Wallet interface {
	Pay(ctx context.Context, payment bonuses.Payment) (bonuses.Paid, error)
}

type Publisher interface {
	Publish(events ...proto.Message)
}

type In struct {
	Caller    bonuses.Caller
	Token     string
	CountryID string
}

type Out struct {
	Kind bonuses.Kind

	// Amount is what the wallet kept.
	Amount int

	Hand bonuses.Hand
}

func New(registry Registry, wallet Wallet, events Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{registry: registry, wallet: wallet, events: events, clock: clock}
}

type UseCase struct {
	registry Registry
	wallet   Wallet
	events   Publisher
	clock    cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	reward, ok := u.registry.Claim(in.Token, bonuses.EntrantOf(in.Caller), in.Caller.Scope)
	if !ok {
		return Out{}, ErrNoSuchBonus
	}

	// The offer is taken: a caller who leaves now must still get the charge.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grantTimeout)
	defer cancel()

	paid, err := u.wallet.Pay(ctx, bonuses.Payment{
		Grant: bonuses.OfferGrantOf(in.Token), Account: in.Caller.Account, Kind: reward.Kind, Amount: reward.Amount,
	})
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the wallet named it.
	}

	u.registry.Publish(bonuses.Taken{CountryID: in.CountryID, Kind: reward.Kind})
	if account := in.Caller.Account; account != bonuses.NoAccount {
		u.events.Publish(&planetv1.BonusClaimed{AccountId: string(account), ClaimedAt: timestamppb.New(u.clock.Now())})
	}

	return Out{Kind: reward.Kind, Amount: paid.Kept, Hand: paid.Hand}, nil
}
