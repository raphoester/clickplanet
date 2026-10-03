// Package record_allegiance_usecase counts a tile taken for the flag of the account and the scope that took it.
package record_allegiance_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Allegiances interface {
	Allegiances(ctx context.Context, keys ...clicks.AllegianceKey) (map[clicks.AllegianceKey]clicks.Allegiance, error)
	SaveAllegiances(ctx context.Context, tallies map[clicks.AllegianceKey]clicks.Allegiance) error
}

type In struct {
	Account string
	Scope   string
	Country string
	At      time.Time
}

func New(allegiances Allegiances) *UseCase {
	return &UseCase{allegiances: allegiances}
}

type UseCase struct {
	allegiances Allegiances
}

// Execute is a read then a save: the subscriber is the one writer, so nothing moves a tally in between.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	keys := clicks.Payer{Scope: in.Scope, Account: in.Account}.AllegianceKeys()

	tallies, err := u.allegiances.Allegiances(ctx, keys...)
	if err != nil {
		return fmt.Errorf("failed to read the tallies: %w", err)
	}

	counted := make(map[clicks.AllegianceKey]clicks.Allegiance, len(keys))
	for _, key := range keys {
		counted[key] = tallies[key].With(in.Country, in.At)
	}

	if err := u.allegiances.SaveAllegiances(ctx, counted); err != nil {
		return fmt.Errorf("failed to save the tallies: %w", err)
	}

	return nil
}
