package rounds

import (
	"cmp"
	"slices"
)

type Score struct {
	Country      Country
	Points       uint64
	RoundsWon    uint32
	FinalePoints uint64
}

type Placed struct {
	Score
	Rank uint32
}

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
		cmp.Compare(b.FinalePoints, a.FinalePoints),
	)
}
