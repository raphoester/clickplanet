package patrol_usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/swastikas"
)

// A painting smaller than the smallest swastika players drew is no drawing worth reading.
const leastTiles = 12

type Planet interface {
	Takes(ctx context.Context, since, until time.Time) ([]paint.Take, error)
	// One tile per id asked, in the order asked.
	Tiles(ctx context.Context, ids []drawings.TileID) ([]drawings.MapTile, error)
}

type Graffiti interface {
	PaintedBy(ctx context.Context, account cpsession.AccountID) ([]graffiti.Graffiti, error)
	Save(ctx context.Context, graffiti ...graffiti.Graffiti) error
}

type Reporter interface {
	Report(ctx context.Context, graffiti graffiti.Graffiti)
}

type Clock interface {
	Now() time.Time
}

func New(planet Planet, kept Graffiti, reporter Reporter, clock Clock, patrol graffiti.Patrol) *UseCase {
	return &UseCase{planet: planet, graffiti: kept, reporter: reporter, clock: clock, patrol: patrol}
}

type UseCase struct {
	planet   Planet
	graffiti Graffiti
	reporter Reporter
	clock    Clock
	patrol   graffiti.Patrol
}

// One look at what each player painted lately: the tiles it still holds, split into drawings, each read, and each
// swastika that shows on the map flagged.
func (u *UseCase) Execute(ctx context.Context) error {
	now := u.clock.Now()
	takes, err := u.planet.Takes(ctx, u.patrol.Since(now), now)
	if err != nil {
		return fmt.Errorf("failed to read what players painted: %w", err)
	}
	paintings := paint.PaintingsOf(takes, leastTiles)
	if len(paintings) == 0 {
		return nil
	}

	var painted []drawings.TileID
	for _, painting := range paintings {
		painted = append(painted, painting.Tiles...)
	}
	tiles, err := u.planet.Tiles(ctx, unique(painted))
	if err != nil {
		return fmt.Errorf("failed to ask planet the tiles players painted: %w", err)
	}
	onMap := make(map[drawings.TileID]drawings.MapTile, len(tiles))
	for _, tile := range tiles {
		onMap[tile.ID] = tile
	}

	var sightings []graffiti.Sighting
	var rims []drawings.TileID
	for _, painting := range paintings {
		for _, drawing := range drawings.Split(painting.Held(onMap)) {
			if _, ok := swastikas.ReadingOf(drawing); ok {
				sightings = append(sightings, graffiti.Sighting{
					Account: painting.Account, Flag: painting.Flag, Kind: graffiti.Swastika, Drawing: drawing, At: now,
				})
				rims = append(rims, drawing.Rim()...)
			}
		}
	}
	if len(sightings) == 0 {
		return nil
	}

	around, err := u.planet.Tiles(ctx, unique(rims))
	if err != nil {
		return fmt.Errorf("failed to ask planet the tiles around the drawings: %w", err)
	}
	flags := make(map[drawings.TileID]string, len(around))
	for _, tile := range around {
		flags[tile.ID] = tile.Flag
	}

	for _, seen := range sightings {
		if err := u.note(ctx, seen, flags); err != nil {
			return err
		}
	}
	return nil
}

func (u *UseCase) note(ctx context.Context, seen graffiti.Sighting, flags map[drawings.TileID]string) error {
	rim := seen.Drawing.Rim()
	aroundFlags := make([]string, len(rim))
	for i, id := range rim {
		aroundFlags[i] = flags[id]
	}
	if !drawings.Visible(seen.Flag, aroundFlags) {
		return nil
	}

	painted, err := u.graffiti.PaintedBy(ctx, seen.Account)
	if err != nil {
		return fmt.Errorf("failed to read the graffiti its painter has: %w", err)
	}
	noted, fresh, err := u.patrol.Note(painted, seen)
	if err != nil {
		return err //nolint:wrapcheck // the patrol named it.
	}
	if err := u.graffiti.Save(ctx, noted); err != nil {
		return fmt.Errorf("failed to keep a graffiti: %w", err)
	}
	if fresh {
		u.reporter.Report(ctx, noted)
	}
	return nil
}

func unique(ids []drawings.TileID) []drawings.TileID {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	return slices.Compact(sorted)
}
