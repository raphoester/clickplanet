package missions

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"
)

type Holding struct {
	Country string
	Tiles   uint32
}

type Territory struct {
	Flag    string
	Tiles   uint32
	Grounds []Holding
}

type World struct {
	Territories  []Territory
	CountrySizes []Holding
}

// A span is what a goal may be, drawn in steps of ten tiles.
type span struct {
	least, most uint32
}

var (
	bankGoal   = span{least: 120, most: 240}
	rivalGoal  = span{least: 80, most: 160}
	groundGoal = span{least: 300, most: 600}
)

const (
	goalStep = 10

	rivalAtLeast  = 1_000
	groundAtLeast = 3_000
	openAtLeast   = 1_000

	fewestMissions = 3
	mostMissions   = 5
)

// A purse is a charge a mission may pay and how much of it: what multiplies or guards clicks, never clicks.
type purse struct {
	kind        RewardKind
	least, most uint32
}

var purses = []purse{
	{kind: RewardShields, least: 15, most: 30},
	{kind: RewardSpreadClicks, least: 4, most: 8},
	{kind: RewardBomb, least: 1, most: 1},
	{kind: RewardEncloseClicks, least: 2, most: 3},
}

func Draw(day Day, account AccountID, flag string, world World) Brief {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(day.String() + "/" + account.String()))
	d := draw{flag: flag, random: rand.New(rand.NewPCG(seed.Sum64(), 0))} //nolint:gosec // a draw every player can see, not a secret.

	orders := d.orders(d.ranked(rivalsOf(flag, world)), groundsOf(flag, world))
	rewards := d.rewards(len(orders))

	missions := make([]Mission, 0, len(orders))
	for n, order := range orders {
		missions = append(missions, Mission{id: drawnID(day, account, n+1), order: order, reward: rewards[n]})
	}
	return Brief{missions: missions}
}

var drawnSpace = uuid.MustParse("4d697373-696f-4e73-8000-6d697373696f")

func drawnID(day Day, account AccountID, n int) MissionID {
	return MissionID(uuid.NewSHA1(drawnSpace, fmt.Appendf(nil, "%s/%s/%d", day, account, n)))
}

// The flags holding rivalAtLeast tiles but the flag, the biggest first.
func rivalsOf(flag string, world World) []Holding {
	var rivals []Holding
	for _, held := range world.Territories {
		if held.Flag != flag && held.Tiles >= rivalAtLeast {
			rivals = append(rivals, Holding{Country: held.Flag, Tiles: held.Tiles})
		}
	}
	slices.SortFunc(rivals, func(a, b Holding) int {
		return cmp.Or(cmp.Compare(b.Tiles, a.Tiles), cmp.Compare(a.Country, b.Country))
	})
	return rivals
}

// The countries of groundAtLeast tiles or more but the flag, with openAtLeast tiles the flag does not hold, by code.
func groundsOf(flag string, world World) []string {
	held := make(map[string]uint32)
	for _, territory := range world.Territories {
		if territory.Flag != flag {
			continue
		}
		for _, ground := range territory.Grounds {
			held[ground.Country] = ground.Tiles
		}
	}

	var grounds []string
	for _, size := range world.CountrySizes {
		if size.Country != flag && size.Tiles >= groundAtLeast && size.Tiles >= held[size.Country]+openAtLeast {
			grounds = append(grounds, size.Country)
		}
	}
	slices.Sort(grounds)
	return grounds
}

type draw struct {
	flag   string
	random *rand.Rand
}

func (d draw) goal(of span) uint32 {
	return of.least + goalStep*d.random.Uint32N((of.most-of.least)/goalStep+1)
}

// Every rival once, each drawn by the ground it holds, so the leader is the likeliest first and never the only one.
func (d draw) ranked(rivals []Holding) []string {
	left := slices.Clone(rivals)
	ranked := make([]string, 0, len(left))
	for len(left) > 0 {
		var total uint64
		for _, rival := range left {
			total += uint64(rival.Tiles)
		}
		drawn := d.random.Uint64N(total)
		n := 0
		for drawn >= uint64(left[n].Tiles) {
			drawn -= uint64(left[n].Tiles)
			n++
		}
		ranked = append(ranked, left[n].Country)
		left = slices.Delete(left, n, n+1)
	}
	return ranked
}

// The bank, a rival and a ground, then more rivals and grounds up to what the day gives.
func (d draw) orders(rivals, grounds []string) []Order {
	count := fewestMissions + d.random.IntN(mostMissions-fewestMissions+1)

	orders := []Order{TakeFor{Flag: d.flag, Goal: d.goal(bankGoal)}}
	if len(rivals) == 0 {
		orders = append(orders, TakeFor{Flag: d.flag, Goal: 2 * d.goal(rivalGoal)})
	} else {
		orders = append(orders, TakeFrom{Flag: rivals[0], Goal: d.goal(rivalGoal)})
	}
	var ground string
	if len(grounds) == 0 {
		orders = append(orders, TakeFor{Flag: d.flag, Goal: d.goal(groundGoal)})
	} else {
		ground = grounds[d.random.IntN(len(grounds))]
		orders = append(orders, TakeIn{Ground: ground, Goal: d.goal(groundGoal)})
	}

	var extras []Order
	if len(rivals) > 1 {
		for _, rival := range rivals[1:] {
			extras = append(extras, TakeFrom{Flag: rival, Goal: d.goal(rivalGoal)})
		}
	}
	for _, other := range grounds {
		if other != ground {
			extras = append(extras, TakeIn{Ground: other, Goal: d.goal(groundGoal)})
		}
	}
	d.random.Shuffle(len(extras), func(i, j int) { extras[i], extras[j] = extras[j], extras[i] })
	return append(orders, extras[:min(count-len(orders), len(extras))]...)
}

// Every charge once before any twice, in a random order, each as much as its purse gives.
func (d draw) rewards(count int) []Reward {
	rewards := make([]Reward, 0, count)
	for len(rewards) < count {
		drawn := slices.Clone(purses)
		d.random.Shuffle(len(drawn), func(i, j int) { drawn[i], drawn[j] = drawn[j], drawn[i] })
		for _, purse := range drawn {
			if len(rewards) == count {
				break
			}
			rewards = append(rewards, Reward{Kind: purse.kind, Amount: purse.least + d.random.Uint32N(purse.most-purse.least+1)})
		}
	}
	return rewards
}
