// Package drop_bomb spends a caller's bomb where they aimed: it clears the tiles around the one hit,
// or, in the sea, nothing.
package drop_bomb

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// ErrNoBomb covers never won, already dropped and held too long, for the reason ErrNoSuchBonus does.
var ErrNoBomb = errors.New("no bomb to drop")

type Bombs interface {
	Take(scope string) bool
}

// Schedule is told a bomb went off, so the next box is not held back by time it was not held.
type Schedule interface {
	Dropped(scope string)
}

type Map interface {
	Nearest(point clicks.Vec3) (uint32, float64)
	Position(id uint32) (clicks.Vec3, bool)
	Within(centre clicks.Vec3, radius float64) []uint32
}

type Clearer interface {
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Target    clicks.Vec3
	CountryID string
}

// Rules is what a bomb is: how wide a circle it clears, and how far from a tile an aim may land and still hit it.
type Rules struct {
	Radius float64
	Reach  float64
}

func New(bombs Bombs, schedule Schedule, geography Map, clearer Clearer, countries CountryChecker, rules Rules) *UseCase {
	return &UseCase{
		bombs:     bombs,
		schedule:  schedule,
		geography: geography,
		clearer:   clearer,
		countries: countries,
		rules:     rules,
	}
}

type UseCase struct {
	bombs     Bombs
	schedule  Schedule
	geography Map
	clearer   Clearer
	countries CountryChecker
	rules     Rules
}

// Execute checks the request before it takes the bomb, so a malformed drop does not cost one.
func (u *UseCase) Execute(ctx context.Context, in In) (clicks.Blast, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return clicks.Blast{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	tile, arc := u.geography.Nearest(in.Target)
	if tile == 0 {
		return clicks.Blast{}, fmt.Errorf("%w: a target with no direction", clicks.ErrTileOutOfRange)
	}

	scope := cpctx.RateLimitKey(ctx)
	if !u.bombs.Take(scope) {
		return clicks.Blast{}, ErrNoBomb
	}
	u.schedule.Dropped(scope)

	blast := clicks.Blast{CountryID: in.CountryID, Radius: u.rules.Radius, Point: unit(in.Target)}

	// Too far from any tile is the sea: the bomb is spent all the same.
	if arc <= u.rules.Reach {
		blast.Tile = tile
		blast.Point, _ = u.geography.Position(tile)
		blast.Cleared = u.geography.Within(blast.Point, u.rules.Radius)
	}

	blast, err := u.clearer.Clear(ctx, blast)
	if err != nil {
		return clicks.Blast{}, fmt.Errorf("failed to clear the blast at tile %d: %w", tile, err)
	}

	return blast, nil
}

func unit(v clicks.Vec3) clicks.Vec3 {
	length := v.X*v.X + v.Y*v.Y + v.Z*v.Z
	if length == 0 {
		return v
	}

	scale := 1 / math.Sqrt(length)

	return clicks.Vec3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}
