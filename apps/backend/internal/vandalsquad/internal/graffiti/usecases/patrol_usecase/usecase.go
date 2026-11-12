package patrol_usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/swastikas"
)

// A painting smaller than the smallest swastika players drew is no drawing worth reading.
const leastTiles = 12

type Window interface {
	Paintings(least int) []paint.Painting
	Forget()
}

// The map as planet holds it now, one tile per id asked, in the order asked.
type Map interface {
	Tiles(ctx context.Context, ids []drawings.TileID) ([]drawings.MapTile, error)
}

type Book interface {
	Flagged(graffiti graffiti.Graffiti) bool
	Keep(graffiti graffiti.Graffiti)
}

// Told of each drawing the first time it is flagged.
type Reporter interface {
	Report(ctx context.Context, graffiti graffiti.Graffiti)
}

type Clock interface {
	Now() time.Time
}

func New(window Window, planet Map, book Book, reporter Reporter, clock Clock, page string) *UseCase {
	return &UseCase{window: window, planet: planet, book: book, reporter: reporter, clock: clock, page: page}
}

type UseCase struct {
	window   Window
	planet   Map
	book     Book
	reporter Reporter
	clock    Clock
	page     string
}

type sighting struct {
	painting paint.Painting
	drawing  drawings.Drawing
}

// One look at what each player painted in the window: the tiles it still holds, split into drawings, each read, and
// each swastika that shows on the map flagged.
func (u *UseCase) Execute(ctx context.Context) error {
	u.window.Forget()
	paintings := u.window.Paintings(leastTiles)
	if len(paintings) == 0 {
		return nil
	}
	now := u.clock.Now()

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

	var sightings []sighting
	var rims []drawings.TileID
	for _, painting := range paintings {
		var held []drawings.Tile
		for _, id := range painting.Tiles {
			// A tile taken since, cleared by a bomb or reverted is no longer the player's drawing.
			if tile, ok := onMap[id]; ok && tile.Holder == painting.Account && tile.Flag == painting.Flag {
				held = append(held, tile.Tile)
			}
		}
		for _, drawing := range drawings.Split(held) {
			if _, ok := swastikas.ReadingOf(drawing); ok {
				sightings = append(sightings, sighting{painting: painting, drawing: drawing})
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
		u.note(ctx, seen, flags, now)
	}
	return nil
}

func (u *UseCase) note(ctx context.Context, seen sighting, flags map[drawings.TileID]string, now time.Time) {
	rim := seen.drawing.Rim()
	aroundFlags := make([]string, len(rim))
	for i, id := range rim {
		aroundFlags[i] = flags[id]
	}
	if !drawings.Visible(seen.painting.Flag, aroundFlags) {
		return
	}

	tiles := seen.drawing.Tiles()
	points := make([]drawings.Point, len(tiles))
	for i, tile := range tiles {
		points[i] = tile.Point
	}
	flagged := graffiti.Graffiti{
		Account:   seen.painting.Account,
		Flag:      seen.painting.Flag,
		Kind:      graffiti.Swastika,
		Tiles:     seen.drawing.IDs(),
		FlaggedAt: now,
		SeenAt:    now,
		Link:      graffiti.LinkOf(u.page, points),
	}
	if !u.book.Flagged(flagged) {
		u.reporter.Report(ctx, flagged)
	}
	u.book.Keep(flagged)
}

func unique(ids []drawings.TileID) []drawings.TileID {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	return slices.Compact(sorted)
}
