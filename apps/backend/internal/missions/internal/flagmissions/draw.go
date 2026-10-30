package flagmissions

import (
	"hash/fnv"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

const (
	fewestPoints = 1
	mostPoints   = 3
	goalStep     = 10
)

// A pace is what a goal asks for its points: a share of what the flag did the day before, and never less than least.
type pace struct {
	percent uint64
	least   uint32
}

func (p pace) goal(yesterday uint64) uint32 {
	share := yesterday * p.percent / 100
	share -= share % goalStep
	return max(p.least, uint32(min(share, uint64(^uint32(0))))) //nolint:gosec // capped just above.
}

var (
	conquerPaces = [mostPoints]pace{{percent: 50, least: 1_000}, {percent: 75, least: 1_500}, {percent: 100, least: 2_000}}
	shieldPaces  = [mostPoints]pace{{percent: 50, least: 300}, {percent: 75, least: 450}, {percent: 100, least: 600}}
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
			Ground: grounds[random.IntN(len(grounds))], Goal: conquerPaces[points-1].goal(yesterday.takes),
		})
	}
	if rivals := world.RivalsOf(flag); len(rivals) > 0 {
		orders = append(orders, BreakShields{Flag: byGround(random, rivals), Goal: shieldPaces[points-1].goal(yesterday.strikes)})
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
