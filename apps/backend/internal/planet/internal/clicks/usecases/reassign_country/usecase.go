// Package reassign_country gives every tile one country holds to another, while the game runs.
package reassign_country

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// Pacing keeps each batch of updates inside what an open stream can buffer.
type Pacing struct {
	Batch int
	Pause time.Duration
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

func New(tiles Map, countries CountryChecker, pacing Pacing) *UseCase {
	return &UseCase{tiles: tiles, countries: countries, pacing: pacing}
}

type UseCase struct {
	tiles     Map
	countries CountryChecker
	pacing    Pacing
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

		if err := pause(ctx, u.pacing.Pause); err != nil {
			return u.settle(out, in), err
		}
	}

	return u.settle(out, in), nil
}

func (u *UseCase) settle(out Out, in In) Out {
	out.FromAfter, out.ToAfter = u.tiles.Held(in.From), u.tiles.Held(in.To)
	return out
}

func pause(ctx context.Context, d time.Duration) error {
	if d > 0 {
		timer := time.NewTimer(d)
		defer timer.Stop()

		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("reassignment interrupted: %w", err)
	}

	return nil
}
