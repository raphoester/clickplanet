package inprocess_title_catalog

import (
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playerread"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

func New(catalog titles.Catalog) Catalog {
	return Catalog{catalog: catalog}
}

type Catalog struct {
	catalog titles.Catalog
}

func (c Catalog) Shown(held []string) []*playerv1.Title {
	return playermessage.Titles(c.catalog.Shown(idsOf(held)))
}

func (c Catalog) Worn(held []string, choice string) *playerv1.Title {
	return playermessage.Title(wearing.ShowcaseOf(c.catalog, idsOf(held), titles.ID(choice)).Worn())
}

func (c Catalog) Tracks(held []string, career playerread.Career) []*playerv1.Track {
	stats := players.StatsOf(
		players.AccountID{}, career.TilesTaken, players.StreakOf(career.StreakNow, players.Day{}), 0, career.MessagesSent,
	)
	progress := c.catalog.Progress(titles.CareerOf(stats, players.AccountOf(false, time.Time{})), idsOf(held))

	tracks := make([]*playerv1.Track, 0, len(progress))
	for _, track := range progress {
		steps := make([]*playerv1.Step, 0, len(track.Steps()))
		for _, step := range track.Steps() {
			steps = append(steps, &playerv1.Step{
				Title: playermessage.Title(step.Standing()), Threshold: step.Threshold(), Earned: step.Earned(),
			})
		}
		tracks = append(tracks, &playerv1.Track{
			Id: string(track.ID()), Name: track.Name(), Progress: track.Progress(), Steps: steps,
		})
	}
	return tracks
}

func idsOf(held []string) titles.IDs {
	ids := make(titles.IDs, 0, len(held))
	for _, id := range held {
		ids = append(ids, titles.ID(id))
	}
	return ids
}
