package flagmissions

import (
	"cmp"
	"hash/fnv"
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

const (
	fewestPoints = 3
	mostPoints   = 10
	goalStep     = 10
	goalPercent  = 30
	leastGoal    = 250
)

// A tile taken and a shield broken are a click each, so both kinds ask a part of all the flag did.
func goalOf(yesterday Day) uint32 {
	goal := max(leastGoal, (yesterday.takes+yesterday.strikes)*goalPercent/100)
	goal -= goal % goalStep
	return uint32(min(goal, uint64(^uint32(0)))) //nolint:gosec // capped just above.
}

// mostPoints for a flag with no tiles, fewestPoints for the biggest, and in proportion between, rounded.
func pointsOf(flag string, world missions.World) uint32 {
	var held, biggest uint64
	for _, territory := range world.Territories {
		biggest = max(biggest, uint64(territory.Tiles))
		if territory.Flag == flag {
			held = uint64(territory.Tiles)
		}
	}
	if biggest == 0 {
		return mostPoints
	}
	return mostPoints - uint32(((mostPoints-fewestPoints)*held*2+biggest)/(2*biggest)) //nolint:gosec // at most 7.
}

var drawnSpace = uuid.MustParse("466c6167-4d69-4373-8000-696f6e66c167")

// One mission for the whole flag: a big ground to take tiles on, or a rival's shields to break, evenly.
func Draw(day days.Day, flag string, world missions.World, yesterday Day) Day {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(day.String() + "/" + flag))
	random := rand.New(rand.NewPCG(seed.Sum64(), 0)) //nolint:gosec // a draw every player can see, not a secret.

	var orders []Order
	if grounds := world.GroundsOf(flag); len(grounds) > 0 {
		orders = append(orders, conquest(random, grounds, goalOf(yesterday)))
	}
	if rivals := world.RivalsOf(flag); len(rivals) > 0 {
		orders = append(orders, BreakShields{Flag: byGround(random, rivals), Goal: goalOf(yesterday)})
	}
	if len(orders) == 0 {
		return Day{}
	}

	id := MissionID(uuid.NewSHA1(drawnSpace, []byte(day.String()+"/"+flag)))
	return Day{mission: &MissionSnapshot{ID: id, Order: orders[random.IntN(len(orders))], Points: pointsOf(flag, world)}}
}

// A ground with room for the goal, the tiles the flag does not hold there, drawn evenly; with none, the ground with the
// most room, and the goal cut to it.
func conquest(random *rand.Rand, grounds []missions.Ground, goal uint32) TakeIn {
	var roomy []missions.Ground
	for _, ground := range grounds {
		if ground.Open >= goal {
			roomy = append(roomy, ground)
		}
	}
	if len(roomy) > 0 {
		return TakeIn{Ground: roomy[random.IntN(len(roomy))].Country, Goal: goal}
	}

	widest := slices.MaxFunc(grounds, func(a, b missions.Ground) int { return cmp.Compare(a.Open, b.Open) })
	return TakeIn{Ground: widest.Country, Goal: widest.Open - widest.Open%goalStep}
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
