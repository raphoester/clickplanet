package flagmissions

import (
	"hash/fnv"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

const (
	fewestPoints = 3
	mostPoints   = 10
	goalStep     = 10
)

// A pace is what a goal asks for its points: a share of what the flag did the day before, and never less than least.
type pace struct {
	percent span
	least   span
}

// From its value for the fewest points to its value for the most, evenly.
type span struct {
	fewest uint64
	most   uint64
}

func (s span) at(points uint32) uint64 {
	return s.fewest + (s.most-s.fewest)*uint64(points-fewestPoints)/(mostPoints-fewestPoints)
}

func (p pace) goal(points uint32, yesterday uint64) uint32 {
	goal := max(p.least.at(points), yesterday*p.percent.at(points)/100)
	goal -= goal % goalStep
	return uint32(min(goal, uint64(^uint32(0)))) //nolint:gosec // capped just above.
}

var (
	conquerPace = pace{percent: span{fewest: 50, most: 100}, least: span{fewest: 1_000, most: 2_000}}
	shieldPace  = pace{percent: span{fewest: 50, most: 100}, least: span{fewest: 300, most: 600}}
)

var drawnSpace = uuid.MustParse("466c6167-4d69-4373-8000-696f6e66c167")

// One mission for the whole flag: a big ground to take tiles on, or a rival's shields to break, evenly; the more
// points, the bigger the share of yesterday it asks.
func Draw(day days.Day, flag string, world missions.World, yesterday Day) Day {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(day.String() + "/" + flag))
	random := rand.New(rand.NewPCG(seed.Sum64(), 0)) //nolint:gosec // a draw every player can see, not a secret.

	points := fewestPoints + random.Uint32N(mostPoints-fewestPoints+1)
	var orders []Order
	if grounds := world.GroundsOf(flag); len(grounds) > 0 {
		orders = append(orders, TakeIn{
			Ground: grounds[random.IntN(len(grounds))], Goal: conquerPace.goal(points, yesterday.takes),
		})
	}
	if rivals := world.RivalsOf(flag); len(rivals) > 0 {
		orders = append(orders, BreakShields{Flag: byGround(random, rivals), Goal: shieldPace.goal(points, yesterday.strikes)})
	}
	if len(orders) == 0 {
		return Day{}
	}

	id := MissionID(uuid.NewSHA1(drawnSpace, []byte(day.String()+"/"+flag)))
	return Day{mission: &MissionSnapshot{ID: id, Order: orders[random.IntN(len(orders))], Points: points}}
}

// A rival drawn by the ground it holds, so the leaders share the heat.
func byGround(random *rand.Rand, rivals []missions.Holding) string {
	var total uint64
	for _, rival := range rivals {
		total += uint64(rival.Tiles)
	}
	drawn := random.Uint64N(total)
	for _, rival := range rivals {
		if drawn < uint64(rival.Tiles) {
			return rival.Country
		}
		drawn -= uint64(rival.Tiles)
	}
	return rivals[len(rivals)-1].Country
}
