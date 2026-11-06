package books

import (
	"cmp"
	"maps"
	"slices"
)

type Result struct {
	Country Country
	Rank    uint32
	Points  uint32
	Share   float64
}

type Scored struct {
	Round   Round
	Results []Result
}

var points = []uint32{25, 18, 15, 12, 10, 8, 6, 4, 2, 1}

// Results ranks the countries holding tiles at the end by their gain since the start; only a gain scores.
func (r Round) Results(start Start, end Snapshot) []Result {
	gains := map[Country]int64{}
	for country, tiles := range end.Held {
		if tiles > 0 {
			gains[country] = int64(tiles) - int64(start[country])
		}
	}
	countries := slices.Collect(maps.Keys(gains))
	slices.SortFunc(countries, func(a, b Country) int {
		return cmp.Or(cmp.Compare(gains[b], gains[a]), cmp.Compare(a, b))
	})

	results := make([]Result, 0, len(countries))
	for i, country := range countries {
		rank := uint32(i + 1) //nolint:gosec // a few hundred countries at most.
		if i > 0 && gains[country] == gains[countries[i-1]] {
			rank = results[i-1].Rank
		}
		won := uint32(0)
		if gains[country] > 0 {
			won = pointsAt(rank, r.Times)
		}
		results = append(results, Result{Country: country, Rank: rank, Points: won, Share: float64(end.Held[country]) / float64(end.Tiles)})
	}
	return results
}

func pointsAt(rank, times uint32) uint32 {
	if int(rank) > len(points) {
		return 0
	}
	return points[rank-1] * times
}

// RankPoints is what a rank of a round scores.
type RankPoints struct {
	Rank   uint32
	Points uint32
}

// RankPointsOf is what each rank of a round of times scores, the first's first: every rank that scores.
func RankPointsOf(times uint32) ([]RankPoints, error) {
	if err := checkTimes(times); err != nil {
		return nil, err
	}
	ranks := make([]RankPoints, 0, len(points))
	for i := range points {
		rank := uint32(i + 1) //nolint:gosec // ten ranks.
		ranks = append(ranks, RankPoints{Rank: rank, Points: pointsAt(rank, times)})
	}
	return ranks, nil
}
