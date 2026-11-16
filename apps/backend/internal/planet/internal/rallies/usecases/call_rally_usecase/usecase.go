package call_rally_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var ErrNoRally = errors.New("no rally to call")

const returnTimeout = 5 * time.Second

type Gate interface {
	Call(ctx context.Context, payer clicks.Payer, account cpsession.AccountID, tile uint32, country string) error
}

type Hands interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
	Return(ctx context.Context, holder charges.Holder, back func(charges.Held) charges.Held) error
}

type Muster interface {
	Rally(account cpsession.AccountID, flag string, tile uint32) (rallies.Rally, error)
}

type Calls interface {
	Change(ctx context.Context, account cpsession.AccountID, change func(rallies.Rally) (rallies.Rally, error)) error
}

type In struct {
	TileID    uint32
	CountryID string
}

func New(gate Gate, hands Hands, muster Muster, calls Calls, teller rallies.Teller) *UseCase {
	return &UseCase{gate: gate, hands: hands, muster: muster, calls: calls, teller: teller}
}

type UseCase struct {
	gate   Gate
	hands  Hands
	muster Muster
	calls  Calls
	teller rallies.Teller
}

// Execute spends a rally once the gate let it through, keeps it as the caller's last, gives it back when a rally of the
// caller turned out to be still on, and tells every stream and the other modules of it.
func (u *UseCase) Execute(ctx context.Context, in In) (rallies.Rally, error) {
	payer := clicks.PayerOf(ctx)
	id, err := uuid.Parse(payer.Account)
	if err != nil {
		return rallies.Rally{}, ErrNoRally
	}
	account := cpsession.AccountID(id)

	if err := u.gate.Call(ctx, payer, account, in.TileID, in.CountryID); err != nil {
		return rallies.Rally{}, err //nolint:wrapcheck // the gate names why.
	}

	holder := charges.HolderOf(payer)
	if err := u.hands.Spend(ctx, holder, charges.Held.AfterRally); err != nil {
		if errors.Is(err, charges.ErrNotHeld) {
			return rallies.Rally{}, ErrNoRally
		}
		return rallies.Rally{}, err //nolint:wrapcheck // the wallet names it.
	}

	rally, err := u.muster.Rally(account, in.CountryID, in.TileID)
	if err == nil {
		err = u.calls.Change(ctx, account, func(last rallies.Rally) (rallies.Rally, error) { return last.Called(rally) })
	}
	if err != nil {
		returning, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnTimeout)
		defer cancel()
		return rallies.Rally{}, errors.Join(fmt.Errorf("failed to call the rally: %w", err),
			u.hands.Return(returning, holder, charges.Held.WithRallyBack))
	}

	u.teller.Called(rally)

	return rally, nil
}
