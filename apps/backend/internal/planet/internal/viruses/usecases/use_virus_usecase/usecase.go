package use_virus_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var ErrNoVirus = errors.New("no virus to use")

const returnTimeout = 5 * time.Second

type Gate interface {
	Aim(ctx context.Context, payer clicks.Payer, tile uint32, country string) error
}

type Hands interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
	Return(ctx context.Context, holder charges.Holder, back func(charges.Held) charges.Held) error
}

type Contagion interface {
	Start(ctx context.Context, virus clicks.Virus, most int, next time.Time) (clicks.Virus, error)
}

type Outbreak interface {
	Most() int
	Next() time.Time
}

type In struct {
	TileID    uint32
	CountryID string
}

func New(gate Gate, ids viruses.IDProvider, hands Hands, contagion Contagion, outbreak Outbreak) *UseCase {
	return &UseCase{gate: gate, ids: ids, hands: hands, contagion: contagion, outbreak: outbreak}
}

type UseCase struct {
	gate      Gate
	ids       viruses.IDProvider
	hands     Hands
	contagion Contagion
	outbreak  Outbreak
}

// Execute spends the virus once the gate let it through, and gives it back when the map refuses the tile after all.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	payer := clicks.PayerOf(ctx)
	if err := u.gate.Aim(ctx, payer, in.TileID, in.CountryID); err != nil {
		return err //nolint:wrapcheck // the gate names why.
	}
	id, err := u.ids.NewID()
	if err != nil {
		return err //nolint:wrapcheck // the provider names it.
	}

	holder := charges.HolderOf(payer)
	if err := u.hands.Spend(ctx, holder, charges.Held.AfterVirus); err != nil {
		if errors.Is(err, charges.ErrNotHeld) {
			return ErrNoVirus
		}
		return err //nolint:wrapcheck // the wallet names it.
	}

	account, err := uuid.Parse(payer.Account)
	if err == nil {
		virus := clicks.NewVirus(id, cpsession.AccountID(account), payer.Scope, in.CountryID, in.TileID)
		_, err = u.contagion.Start(ctx, virus, u.outbreak.Most(), u.outbreak.Next())
	}
	if err != nil {
		returning, cancel := context.WithTimeout(context.WithoutCancel(ctx), returnTimeout)
		defer cancel()
		return errors.Join(fmt.Errorf("failed to start the virus: %w", err),
			u.hands.Return(returning, holder, charges.Held.WithVirusBack))
	}

	return nil
}
