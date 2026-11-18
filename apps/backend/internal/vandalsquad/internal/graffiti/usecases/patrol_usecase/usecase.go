package patrol_usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/swastikas"
)

// A painting smaller than the smallest swastika players drew is no drawing worth reading.
const leastTiles = 12

type Planet interface {
	// Every take of the world's map on the board it plays, and that board.
	Takes(ctx context.Context, since, until time.Time) (drawings.Board, []paint.Take, error)
	// One tile per id asked of the board, in the order asked: drawings.ErrOtherBoard when the map in play plays another.
	Tiles(ctx context.Context, board drawings.Board, ids []drawings.TileID) ([]drawings.MapTile, error)
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
	board, takes, err := u.planet.Takes(ctx, u.patrol.Since(now), now)
	if err != nil {
		return fmt.Errorf("failed to read what players painted: %w", err)
	}
	paintings := paint.PaintingsOf(takes, leastTiles)
	if len(paintings) == 0 {
		return nil
	}
	// The largest drawing is as much ground on any board.
	scale, err := cpboards.ScaleOf(string(board))
	if err != nil {
		return fmt.Errorf("failed to scale the drawings to planet's board: %w", err)
	}

	var painted []drawings.TileID
	for _, painting := range paintings {
		painted = append(painted, painting.Tiles...)
	}
	tiles, err := u.planet.Tiles(ctx, board, unique(painted))
	if errors.Is(err, drawings.ErrOtherBoard) {
		// An arena plays a board of its own: the world's drawings wait for the world's map.
		return nil
	}
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
			if _, ok := swastikas.ReadingOf(drawing, scale); ok {
				sightings = append(sightings, graffiti.Sighting{
					Board: board, Account: painting.Account, Flag: painting.Flag, Kind: graffiti.Swastika, Drawing: drawing, At: now,
				})
				rims = append(rims, drawing.Rim()...)
			}
		}
	}
	if len(sightings) == 0 {
		return nil
	}

	around, err := u.planet.Tiles(ctx, board, unique(rims))
	if errors.Is(err, drawings.ErrOtherBoard) {
		return nil
	}
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
