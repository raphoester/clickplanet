// Package paint_random_tiles_usecase paints random tiles with a flag, seeded on one country's ground, while the game runs.
package paint_random_tiles_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	ErrInvalidCount     = errors.New("count must be positive")
	ErrInvalidProximity = errors.New("proximity must be between 0 and 1")
)

type Borders interface {
	CountryOf(tile uint32) string
	Tiles() uint32
}

type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type Map interface {
	Owner(tile uint32) (string, bool)
	Restore(ctx context.Context, restorations []clicks.Restoration) (int, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Flag      string
	Area      string
	Count     int
	Proximity float64
	DryRun    bool
}

type Out struct {
	// Eligible is every tile of the area not wearing the flag yet.
	Eligible    int
	Picked      int
	OutsideArea int
	Painted     int
}

func New(
	borders Borders,
	neighbours Neighbours,
	tiles Map,
	countries CountryChecker,
	random clicks.Random,
	pace clicks.Pacing,
) *UseCase {
	return &UseCase{
		borders:    borders,
		neighbours: neighbours,
		tiles:      tiles,
		countries:  countries,
		random:     random,
		pacing:     pace,
	}
}

type UseCase struct {
	borders    Borders
	neighbours Neighbours
	tiles      Map
	countries  CountryChecker
	random     clicks.Random
	pacing     clicks.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if !u.countries.CheckCountry(in.Flag) {
		return Out{}, fmt.Errorf("%w: flag %q", clicks.ErrUnknownCountry, in.Flag)
	}
	if !u.countries.CheckCountry(in.Area) {
		return Out{}, fmt.Errorf("%w: area %q", clicks.ErrUnknownCountry, in.Area)
	}
	if in.Count <= 0 {
		return Out{}, fmt.Errorf("%w: %d", ErrInvalidCount, in.Count)
	}
	if in.Proximity < 0 || in.Proximity > 1 {
		return Out{}, fmt.Errorf("%w: %v", ErrInvalidProximity, in.Proximity)
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("paint batch must be positive")
	}

	// The owner each tile held when it was judged eligible, so the paint is a compare-and-set against it.
	owners := map[uint32]string{}
	eligible := func(tile uint32) bool {
		owner, _ := u.tiles.Owner(tile)
		if owner == in.Flag {
			return false
		}
		owners[tile] = owner
		return true
	}

	var seeds []uint32
	for tile := uint32(1); tile <= u.borders.Tiles(); tile++ {
		if u.borders.CountryOf(tile) == in.Area && eligible(tile) {
			seeds = append(seeds, tile)
		}
	}

	picked := clicks.Pick(seeds, in.Count, in.Proximity, u.neighbours.Neighbours, eligible, u.random)

	out := Out{Eligible: len(seeds), Picked: len(picked)}
	for _, tile := range picked {
		if u.borders.CountryOf(tile) != in.Area {
			out.OutsideArea++
		}
	}
	if in.DryRun {
		return out, nil
	}

	restorations := make([]clicks.Restoration, 0, len(picked))
	for _, tile := range picked {
		restorations = append(restorations, clicks.Restoration{Tile: tile, From: owners[tile], To: in.Flag})
	}

	for len(restorations) > 0 {
		batch := restorations[:min(u.pacing.Batch, len(restorations))]
		restorations = restorations[len(batch):]

		painted, err := u.tiles.Restore(ctx, batch)
		out.Painted += painted
		if err != nil {
			return out, fmt.Errorf("failed to paint tiles: %w", err)
		}

		if len(restorations) > 0 {
			if err := u.pacing.Wait(ctx); err != nil {
				return out, fmt.Errorf("paint %w", err)
			}
		}
	}

	return out, nil
}
