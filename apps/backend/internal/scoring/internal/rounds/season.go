package rounds

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type SeasonEnd struct {
	Season  calendar.Number
	EndedAt time.Time
	Table   []Placed
}

// SeasonEndOf is a season's end, ranked from every round it closed. It ends at its end, whether a round ends with it or
// none does.
func SeasonEndOf(season calendar.Season, closed []Closed) SeasonEnd {
	return SeasonEnd{Season: season.Number, EndedAt: season.EndsAt, Table: TableOf(ScoresOf(closed))}
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
