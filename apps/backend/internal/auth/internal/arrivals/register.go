package arrivals

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Saver interface {
	Save(ctx context.Context, arrivals ...Arrival) error
}

func NewRegister(store Saver, ids IDProvider, countries Countries) *Register {
	return &Register{store: store, ids: ids, countries: countries}
}

// Where an account was made, it is told how the account came.
type Register struct {
	store     Saver
	ids       IDProvider
	countries Countries
}

func (r *Register) Record(ctx context.Context, account AccountID, landing Landing, now time.Time) error {
	id, err := r.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to get an arrival id: %w", err)
	}
	arrival, err := NewArrival(id, account, landing, r.countries, now)
	if err != nil {
		return err
	}
	if err := r.store.Save(ctx, arrival); err != nil {
		return fmt.Errorf("failed to save the arrival: %w", err)
	}
	return nil
}

type GuestStarter interface {
	Start(ctx context.Context, now time.Time) (accounts.Entry, error)
}

func NewGuests(transactioner cpatomic.Transactioner, guests GuestStarter, register *Register) *Guests {
	return &Guests{transactioner: transactioner, guests: guests, register: register}
}

// A guest started with its arrival, or neither.
type Guests struct {
	transactioner cpatomic.Transactioner
	guests        GuestStarter
	register      *Register
}

func (g *Guests) Start(ctx context.Context, landing Landing, now time.Time) (accounts.Entry, error) {
	var entry accounts.Entry
	err := g.transactioner.InTx(ctx, func(ctx context.Context) error {
		started, err := g.guests.Start(ctx, now)
		if err != nil {
			return fmt.Errorf("failed to start a guest: %w", err)
		}
		if err := g.register.Record(ctx, started.Account(), landing, now); err != nil {
			return fmt.Errorf("failed to record the guest's arrival: %w", err)
		}
		entry = started
		return nil
	})
	return entry, err
}
