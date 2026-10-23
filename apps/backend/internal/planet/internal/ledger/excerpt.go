package ledger

import (
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Every change the live stream showed from since until until, oldest first, and who made it.
func NewExcerpt(since, until time.Time) *Excerpt {
	return &Excerpt{since: since, until: until}
}

type Excerpt struct {
	since time.Time
	until time.Time
	lines []Line
}

type Line struct {
	At     time.Time
	By     Caller
	Update clicks.TileUpdate
}

func (e *Excerpt) See(event Event) {
	by := event.By()

	event.Show(func(scene Scene) {
		if scene.At.Before(e.since) || !scene.At.Before(e.until) {
			return
		}

		switch {
		case scene.Change != nil:
			e.lines = append(e.lines, Line{At: scene.At, By: by, Update: scene.Change.TileUpdate})
		case scene.Blast != nil:
			// A shield the bomb struck changed no hands, and the blast does not say whose tile it stood on.
			for i, tile := range scene.Blast.Cleared {
				e.lines = append(e.lines, Line{
					At: scene.At, By: by, Update: clicks.TileUpdate{Tile: tile, Previous: scene.Blast.Owners[i]},
				})
			}
		case scene.Spread != nil, scene.Enclosure != nil:
			// The tiles a bonus took are Changes of their own.
		}
	})
}

func (e *Excerpt) Lines() []Line {
	return slices.Clone(e.lines)
}
