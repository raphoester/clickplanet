// Package boost_click tells the planet about a click made under a triple
// clicks bonus. The bonus itself is the limiter's: this only announces it.
package boost_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// Boosts says whether a caller's allowance is boosted right now.
type Boosts interface {
	Boosted(key string) bool
}

// Publisher tells the planet a click was boosted, so every client can show it.
type Publisher interface {
	PublishBoosted(boosted bonus.Boosted)
}

func New(implementation click.IUseCase, boosts Boosts, publisher Publisher) *UseCase {
	return &UseCase{implementation: implementation, boosts: boosts, publisher: publisher}
}

type UseCase struct {
	implementation click.IUseCase
	boosts         Boosts
	publisher      Publisher
}

// Execute announces only a click the rule accepted, after it is written.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !u.boosts.Boosted(cpctx.RateLimitKey(ctx)) {
		return out, err
	}

	u.publisher.PublishBoosted(bonus.Boosted{CountryID: in.CountryID, Tile: in.TileID})

	return out, nil
}
