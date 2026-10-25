package rounds

import (
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type SeasonEnd struct {
	Season  calendar.Number
	EndedAt time.Time
	Table   []Placed
}

func (c Closed) SeasonEnd(earlier []Closed) (SeasonEnd, bool) {
	if !c.Round.Finale {
		return SeasonEnd{}, false
	}
	return SeasonEnd{
		Season:  c.Round.Season,
		EndedAt: c.Round.EndsAt,
		Table:   TableOf(ScoresOf(append(slices.Clone(earlier), c))),
	}, true
}

func ScoresOf(season []Closed) []Score {
	scores := map[Country]Score{}
	for _, closed := range season {
		for _, result := range closed.Results {
			if result.Points == 0 {
				continue
			}
			score := scores[result.Country]
			score.Country = result.Country
			score.Points += uint64(result.Points)
			if result.Rank == 1 {
				score.RoundsWon++
			}
			if closed.Round.Finale {
				score.FinalePoints += uint64(result.Points)
			}
			scores[result.Country] = score
		}
	}

	listed := make([]Score, 0, len(scores))
	for _, score := range scores {
		listed = append(listed, score)
	}
	return listed
}

func EndOf(season []Closed) (SeasonEnd, bool) {
	if len(season) == 0 {
		return SeasonEnd{}, false
	}
	last := len(season) - 1
	return season[last].SeasonEnd(season[:last])
}

func (e SeasonEnd) Won(country Country) bool {
	for _, placed := range e.Table {
		if placed.Country == country {
			return placed.Rank == 1
		}
	}
	return false
}
