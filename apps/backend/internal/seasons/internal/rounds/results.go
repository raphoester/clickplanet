package rounds

import (
	"cmp"
	"slices"
)

type Result struct {
	Country Country
	Rank    uint32
	Points  uint32
}

var points = []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1}

const finaleTimes = 3

func (r Round) Results(held map[Country]uint64) []Result {
	countries := make([]Country, 0, len(held))
	for country, tiles := range held {
		if tiles > 0 {
			countries = append(countries, country)
		}
	}
	slices.SortFunc(countries, func(a, b Country) int {
		return cmp.Or(cmp.Compare(held[b], held[a]), cmp.Compare(a, b))
	})

	results := make([]Result, 0, len(countries))
	for i, country := range countries {
		rank := uint32(i + 1) //nolint:gosec // a few hundred countries at most.
		if i > 0 && held[country] == held[countries[i-1]] {
			rank = results[i-1].Rank
		}
		results = append(results, Result{Country: country, Rank: rank, Points: r.pointsAt(rank)})
	}
	return results
}

func (r Round) pointsAt(rank uint32) uint32 {
	if int(rank) > len(points) {
		return 0
	}
	if r.Finale {
		return points[rank-1] * finaleTimes
	}
	return points[rank-1]
}
