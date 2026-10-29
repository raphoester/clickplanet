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

// A span is what a goal may be, drawn in its steps.
type span struct {
	least, most, step uint32
}

var (
	bankGoal    = span{least: 120, most: 240, step: 10}
	rivalGoal   = span{least: 80, most: 160, step: 10}
	groundGoal  = span{least: 300, most: 600, step: 10}
	raidGoal    = span{least: 40, most: 80, step: 10}
	boxGoal     = span{least: 3, most: 6, step: 1}
	answerGoal  = span{least: 2, most: 4, step: 1}
	bombGoal    = span{least: 1, most: 2, step: 1}
	wallGoal    = span{least: 1, most: 3, step: 1}
	fortifyGoal = span{least: 20, most: 60, step: 10}
)

const (
	rivalAtLeast  = 1_000
	groundAtLeast = 3_000
	openAtLeast   = 1_000
	raidAtLeast   = 300

	// planet's bonuses.shield.perTile: a full tile.
	wallShields = 10

	fewestMissions = 4
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

	grounds := world.GroundsOf(flag)
	orders := d.takes(d.ranked(world.RivalsOf(flag)), grounds)
	count := fewestMissions + d.random.IntN(mostMissions-fewestMissions+1)
	orders = append(orders, d.feats(grounds, raidsOf(flag, world), count-len(orders))...)
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
func (w World) RivalsOf(flag string) []Holding {
	var rivals []Holding
	for _, held := range w.Territories {
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
func (w World) GroundsOf(flag string) []string {
	held := make(map[string]uint32)
	for _, territory := range w.Territories {
		if territory.Flag != flag {
			continue
		}
		for _, ground := range territory.Grounds {
			held[ground.Country] = ground.Tiles
		}
	}

	var grounds []string
	for _, size := range w.CountrySizes {
		if size.Country != flag && size.Tiles >= groundAtLeast && size.Tiles >= held[size.Country]+openAtLeast {
			grounds = append(grounds, size.Country)
		}
	}
	slices.Sort(grounds)
	return grounds
}

type raid struct {
	flag, ground string
}

// Every ground on which a rival holds raidAtLeast tiles, by rival and ground.
func raidsOf(flag string, world World) []raid {
	var raids []raid
	for _, held := range world.Territories {
		if held.Flag == flag || held.Tiles < rivalAtLeast {
			continue
		}
		for _, ground := range held.Grounds {
			if ground.Tiles >= raidAtLeast {
				raids = append(raids, raid{flag: held.Flag, ground: ground.Country})
			}
		}
	}
	slices.SortFunc(raids, func(a, b raid) int {
		return cmp.Or(cmp.Compare(a.flag, b.flag), cmp.Compare(a.ground, b.ground))
	})
	return raids
}

type draw struct {
	flag   string
	random *rand.Rand
}

func (d draw) goal(of span) uint32 {
	return of.least + of.step*d.random.Uint32N((of.most-of.least)/of.step+1)
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

// The bank, a rival and a ground.
func (d draw) takes(rivals, grounds []string) []Order {
	orders := []Order{TakeFor{Flag: d.flag, Goal: d.goal(bankGoal)}}
	if len(rivals) == 0 {
		orders = append(orders, TakeFor{Flag: d.flag, Goal: 2 * d.goal(rivalGoal)})
	} else {
		orders = append(orders, TakeFrom{Flag: rivals[0], Goal: d.goal(rivalGoal)})
	}
	if len(grounds) == 0 {
		return append(orders, TakeFor{Flag: d.flag, Goal: d.goal(groundGoal)})
	}
	return append(orders, TakeIn{Ground: grounds[d.random.IntN(len(grounds))], Goal: d.goal(groundGoal)})
}

// Then what the day gives of the other kinds, each once at most, drawn evenly.
func (d draw) feats(grounds []string, raids []raid, count int) []Order {
	feats := []Order{
		CatchBoxes{Goal: d.goal(boxGoal)},
		AnswerQuizzes{Goal: d.goal(answerGoal)},
		ShieldTiles{Shields: wallShields, Goal: d.goal(wallGoal)},
		FortifyTiles{Goal: d.goal(fortifyGoal)},
	}
	if len(grounds) > 0 {
		feats = append(feats, BombIn{Ground: grounds[d.random.IntN(len(grounds))], Goal: d.goal(bombGoal)})
	}
	if len(raids) > 0 {
		drawn := raids[d.random.IntN(len(raids))]
		feats = append(feats, TakeFromIn{Flag: drawn.flag, Ground: drawn.ground, Goal: d.goal(raidGoal)})
	}
	d.random.Shuffle(len(feats), func(i, j int) { feats[i], feats[j] = feats[j], feats[i] })
	return feats[:count]
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
