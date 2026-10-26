package missions

import (
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

const (
	bankGoal   = 60
	rivalGoal  = 40
	groundGoal = 200

	rivalAtLeast  = 400
	groundAtLeast = 1000
)

func Draw(day Day, flag string, world World) Brief {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(day.String() + "/" + flag))
	random := rand.New(rand.NewPCG(seed.Sum64(), 0)) //nolint:gosec // a draw every player can see, not a secret.

	return Brief{missions: []Mission{
		{id: drawnID(day, flag, 1), order: TakeFor{Flag: flag, Goal: bankGoal}, reward: Reward{Kind: RewardShields, Amount: 10}},
		rivalOf(drawnID(day, flag, 2), flag, world),
		groundOf(drawnID(day, flag, 3), flag, world, random),
	}}
}

var drawnSpace = uuid.MustParse("4d697373-696f-4e73-8000-6d697373696f")

// The same draw names its missions the same, so two racing draws of one day agree.
func drawnID(day Day, flag string, n int) MissionID {
	return MissionID(uuid.NewSHA1(drawnSpace, fmt.Appendf(nil, "%s/%s/%d", day, flag, n)))
}

func rivalOf(id MissionID, flag string, world World) Mission {
	reward := Reward{Kind: RewardSpreadClicks, Amount: 4}

	var rival Holding
	for _, held := range world.Territories {
		if held.Country != flag && (held.Tiles > rival.Tiles || held.Tiles == rival.Tiles && held.Country < rival.Country) {
			rival = held
		}
	}
	if rival.Tiles < rivalAtLeast {
		return Mission{id: id, order: TakeFor{Flag: flag, Goal: 2 * bankGoal}, reward: reward}
	}
	return Mission{id: id, order: TakeFrom{Flag: rival.Country, Goal: rivalGoal}, reward: reward}
}

func groundOf(id MissionID, flag string, world World, random *rand.Rand) Mission {
	reward := Reward{Kind: RewardBomb, Amount: 1}

	var grounds []string
	for _, size := range world.CountrySizes {
		if size.Country != flag && size.Tiles >= groundAtLeast {
			grounds = append(grounds, size.Country)
		}
	}
	if len(grounds) == 0 {
		return Mission{id: id, order: TakeFor{Flag: flag, Goal: groundGoal}, reward: reward}
	}
	slices.Sort(grounds)
	return Mission{id: id, order: TakeIn{Ground: grounds[random.IntN(len(grounds))], Goal: groundGoal}, reward: reward}
}
