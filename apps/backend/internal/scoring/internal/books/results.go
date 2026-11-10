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

// Results places the countries holding tiles at the end by the round's ranking; a shared place shares its points.
func (r Round) Results(start Start, end Snapshot) []Result {
	held, gains := map[Country]int64{}, map[Country]int64{}
	for country, tiles := range end.Held {
		if tiles > 0 {
			held[country], gains[country] = int64(tiles), int64(tiles)-int64(start[country])
		}
	}
	heldRanks, gainRanks := ranksOf(held), ranksOf(gains)
	places := make(map[Country]place, len(held))
	for country := range held {
		places[country] = r.Ranking.place(heldRanks[country], gainRanks[country])
	}
	countries := slices.Collect(maps.Keys(places))
	slices.SortFunc(countries, func(a, b Country) int {
		first, second := places[a], places[b]
		return cmp.Or(slices.Compare(first[:], second[:]), cmp.Compare(a, b))
	})

	results := make([]Result, 0, len(countries))
	for i, country := range countries {
		rank := uint32(i + 1) //nolint:gosec // a few hundred countries at most.
		if i > 0 && places[country] == places[countries[i-1]] {
			rank = results[i-1].Rank
		}
		won := uint32(0)
		if r.Ranking.scores(gains[country]) {
			won = pointsAt(rank, r.Times)
		}
		results = append(results, Result{Country: country, Rank: rank, Points: won, Share: float64(end.Held[country]) / float64(end.Tiles)})
	}
	return results
}

func ranksOf(values map[Country]int64) map[Country]uint32 {
	ranks := make(map[Country]uint32, len(values))
	for country, value := range values {
		rank := uint32(1)
		for _, other := range values {
			if other > value {
				rank++
			}
		}
		ranks[country] = rank
	}
	return ranks
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
