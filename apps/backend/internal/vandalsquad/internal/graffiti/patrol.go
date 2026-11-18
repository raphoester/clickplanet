package graffiti

import (
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

type IDProvider interface {
	NewID() (ID, error)
}

type Patrol struct {
	span time.Duration
	page string
	ids  IDProvider
}

func NewPatrol(span time.Duration, page string, ids IDProvider) Patrol {
	return Patrol{span: span, page: page, ids: ids}
}

func (p Patrol) Since(now time.Time) time.Time {
	return now.Add(-p.span)
}

// What a sighting makes of the graffiti its painter has: the one it is seen again as, or a new one, which fresh says.
func (p Patrol) Note(painted []Graffiti, seen Sighting) (noted Graffiti, fresh bool, err error) {
	scale, err := cpboards.ScaleOf(string(seen.Board))
	if err != nil {
		return Graffiti{}, false, fmt.Errorf("failed to frame a drawing on its board: %w", err)
	}
	link := LinkOf(p.page, pointsOf(seen.Drawing), scale)
	if i := slices.IndexFunc(painted, func(g Graffiti) bool { return g.shows(seen) }); i >= 0 {
		return painted[i].seenAgain(seen, link), false, nil
	}

	id, err := p.ids.NewID()
	if err != nil {
		return Graffiti{}, false, fmt.Errorf("failed to name a new graffiti: %w", err)
	}
	noted, err = Snapshot{
		ID: id, Board: seen.Board, Account: seen.Account, Flag: seen.Flag, Kind: seen.Kind, Tiles: seen.Drawing.IDs(),
		FlaggedAt: seen.At, SeenAt: seen.At, Link: link,
	}.Restore()
	if err != nil {
		return Graffiti{}, false, err
	}
	return noted, true, nil
}

func pointsOf(drawing drawings.Drawing) []drawings.Point {
	tiles := drawing.Tiles()
	points := make([]drawings.Point, len(tiles))
	for i, tile := range tiles {
		points[i] = tile.Point
	}
	return points
}
