package books

import (
	"cmp"
	"slices"
)

type Score struct {
	Country        Country
	Points         uint64
	RoundsWon      uint32
	TiebreakPoints uint64
}

type Placed struct {
	Score
	Rank uint32
}

// ScoresOf adds up the results of the rounds scored and the grants, every country with points; tiebreak's points are
// the third key. A grant wins no round.
func ScoresOf(scored []Scored, grants []Grant, tiebreak RoundID) []Score {
	scores := map[Country]Score{}
	for _, round := range scored {
		for _, result := range round.Results {
			if result.Points == 0 {
				continue
			}
			score := scores[result.Country]
			score.Country = result.Country
			score.Points += uint64(result.Points)
			if result.Rank == 1 {
				score.RoundsWon++
			}
			if tiebreak != "" && round.Round.ID == tiebreak {
				score.TiebreakPoints += uint64(result.Points)
			}
			scores[result.Country] = score
		}
	}
	for _, grant := range grants {
		score := scores[grant.Country]
		score.Country = grant.Country
		score.Points += uint64(grant.Points)
		scores[grant.Country] = score
	}

	listed := make([]Score, 0, len(scores))
	for _, score := range scores {
		listed = append(listed, score)
	}
	return listed
}

// TableOf ranks by points, then rounds won, then the tiebreak round's points; countries level on all three share the
// rank.
func TableOf(scores []Score) []Placed {
	sorted := slices.Clone(scores)
	slices.SortFunc(sorted, func(a, b Score) int {
		return cmp.Or(ahead(a, b), cmp.Compare(a.Country, b.Country))
	})

	table := make([]Placed, 0, len(sorted))
	for i, score := range sorted {
		rank := uint32(i + 1) //nolint:gosec // a few hundred countries at most.
		if i > 0 && ahead(sorted[i-1], score) == 0 {
			rank = table[i-1].Rank
		}
		table = append(table, Placed{Score: score, Rank: rank})
	}
	return table
}

func ahead(a, b Score) int {
	return cmp.Or(
		cmp.Compare(b.Points, a.Points),
		cmp.Compare(b.RoundsWon, a.RoundsWon),
		cmp.Compare(b.TiebreakPoints, a.TiebreakPoints),
	)
}
