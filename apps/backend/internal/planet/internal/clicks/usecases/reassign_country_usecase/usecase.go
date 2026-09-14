// Package reassign_country_usecase gives every tile one country holds to another, while the game runs.
package reassign_country_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrSameCountry = errors.New("cannot reassign a country to itself")

type Map interface {
	Held(country string) int
	Reassign(ctx context.Context, from, to string, start uint32, limit int) (next uint32, moved int, err error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	From   string
	To     string
	DryRun bool
}

type Out struct {
	FromBefore int
	ToBefore   int
	Moved      int
	FromAfter  int
	ToAfter    int
}

func New(tiles Map, countries CountryChecker, pace clicks.Pacing) *UseCase {
	return &UseCase{tiles: tiles, countries: countries, pacing: pace}
}

type UseCase struct {
	tiles     Map
	countries CountryChecker
	pacing    clicks.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	for _, country := range []string{in.From, in.To} {
		if !u.countries.CheckCountry(country) {
			return Out{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, country)
		}
	}
	if in.From == in.To {
		return Out{}, fmt.Errorf("%w: %q", ErrSameCountry, in.From)
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("reassign batch must be positive")
	}

	out := Out{FromBefore: u.tiles.Held(in.From), ToBefore: u.tiles.Held(in.To)}
	if in.DryRun {
		out.FromAfter, out.ToAfter = out.FromBefore, out.ToBefore
		return out, nil
	}

	var start uint32
	for {
		next, moved, err := u.tiles.Reassign(ctx, in.From, in.To, start, u.pacing.Batch)
		out.Moved += moved
		if err != nil {
			return u.settle(out, in), fmt.Errorf("failed to reassign tiles: %w", err)
		}
		if next == 0 {
			break
		}
		start = next

		if err := u.pacing.Wait(ctx); err != nil {
			return u.settle(out, in), fmt.Errorf("reassignment %w", err)
		}
	}

	return u.settle(out, in), nil
}

func (u *UseCase) settle(out Out, in In) Out {
	out.FromAfter, out.ToAfter = u.tiles.Held(in.From), u.tiles.Held(in.To)
	return out
}
