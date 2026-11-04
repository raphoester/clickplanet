package inprocess_title_catalog

import (
	"time"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionread"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing"
)

func New(catalog titles.Catalog) Catalog {
	return Catalog{catalog: catalog}
}

type Catalog struct {
	catalog titles.Catalog
}

func (c Catalog) Shown(held []string) []*progressionv1.Title {
	return progressionmessage.Titles(c.catalog.Shown(idsOf(held)))
}

func (c Catalog) Worn(held []string, choice string) *progressionv1.Title {
	return progressionmessage.Title(wearing.ShowcaseOf(c.catalog, idsOf(held), titles.ID(choice)).Worn())
}

func (c Catalog) Tracks(held []string, career progressionread.Career) []*progressionv1.Track {
	stats := stats.StatsSnapshot{
		TilesTaken:     career.TilesTaken,
		Streak:         stats.StreakOf(career.StreakNow, stats.Day{}),
		MessagesSent:   career.MessagesSent,
		BombsDropped:   career.BombsDropped,
		TilesFortified: career.TilesFortified,
		BonusesClaimed: career.BonusesClaimed,
		RightAnswers:   career.RightAnswers,
		MissionsDone:   career.MissionsDone,
	}.Restore()
	progress := c.catalog.Progress(titles.CareerOf(stats, accounts.AccountOf(false, time.Time{}), finales.Fights{}), idsOf(held))

	tracks := make([]*progressionv1.Track, 0, len(progress))
	for _, track := range progress {
		steps := make([]*progressionv1.Step, 0, len(track.Steps()))
		for _, step := range track.Steps() {
			steps = append(steps, &progressionv1.Step{
				Title: progressionmessage.Title(step.Standing()), Threshold: step.Threshold(), Earned: step.Earned(),
			})
		}
		tracks = append(tracks, &progressionv1.Track{
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
