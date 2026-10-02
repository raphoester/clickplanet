package players

import "slices"

type Title string

const (
	Settler   Title = "settler"
	Governor  Title = "governor"
	Conqueror Title = "conqueror"
	Emperor   Title = "emperor"
	Loyal     Title = "loyal"
	Devoted   Title = "devoted"
	Unbroken  Title = "unbroken"
)

type requirement struct {
	title  Title
	tiles  uint64
	streak uint32
}

var ladder = []requirement{
	{title: Settler, tiles: 100},
	{title: Governor, tiles: 1_000},
	{title: Conqueror, tiles: 10_000},
	{title: Emperor, tiles: 100_000},
	{title: Loyal, streak: 7},
	{title: Devoted, streak: 30},
	{title: Unbroken, streak: 100},
}

func (r requirement) metBy(stats Stats) bool {
	return stats.TilesTaken >= r.tiles && stats.StreakBest >= r.streak
}

type Titles []Title

func TitlesOf(stats Stats) Titles {
	var earned Titles
	for _, requirement := range ladder {
		if requirement.metBy(stats) {
			earned = append(earned, requirement.title)
		}
	}
	return earned
}

func (t Titles) Without(held Titles) Titles {
	return slices.DeleteFunc(slices.Clone(t), func(title Title) bool { return slices.Contains(held, title) })
}

func (t Titles) Sorted() Titles {
	var sorted Titles
	for _, requirement := range ladder {
		if slices.Contains(t, requirement.title) {
			sorted = append(sorted, requirement.title)
		}
	}
	return sorted
}
