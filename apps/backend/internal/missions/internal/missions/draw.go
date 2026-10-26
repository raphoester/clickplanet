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

type World struct {
	Territories  []Holding
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

func Draw(day Day, flag string, world World) Brief {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(day.String() + "/" + flag))
	d := draw{flag: flag, random: rand.New(rand.NewPCG(seed.Sum64(), 0))} //nolint:gosec // a draw every player can see, not a secret.

	orders := d.orders(rivalsOf(flag, world), groundsOf(flag, world))
	rewards := d.rewards(len(orders))

	missions := make([]Mission, 0, len(orders))
	for n, order := range orders {
		missions = append(missions, Mission{id: drawnID(day, flag, n+1), order: order, reward: rewards[n]})
	}
	return Brief{missions: missions}
}

var drawnSpace = uuid.MustParse("4d697373-696f-4e73-8000-6d697373696f")

func drawnID(day Day, flag string, n int) MissionID {
	return MissionID(uuid.NewSHA1(drawnSpace, fmt.Appendf(nil, "%s/%s/%d", day, flag, n)))
}

// The flags holding rivalAtLeast tiles but the flag, the biggest first.
func rivalsOf(flag string, world World) []string {
	var rivals []Holding
	for _, held := range world.Territories {
		if held.Country != flag && held.Tiles >= rivalAtLeast {
			rivals = append(rivals, held)
		}
	}
	slices.SortFunc(rivals, func(a, b Holding) int {
		return cmp.Or(cmp.Compare(b.Tiles, a.Tiles), cmp.Compare(a.Country, b.Country))
	})

	flags := make([]string, 0, len(rivals))
	for _, rival := range rivals {
		flags = append(flags, rival.Country)
	}
	return flags
}

// The countries of groundAtLeast tiles or more but the flag, by code.
func groundsOf(flag string, world World) []string {
	var grounds []string
	for _, size := range world.CountrySizes {
		if size.Country != flag && size.Tiles >= groundAtLeast {
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

// The bank, the leader and a ground, then more rivals and grounds up to what the day gives.
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
