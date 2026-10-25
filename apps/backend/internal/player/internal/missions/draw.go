package missions

import (
	"hash/fnv"
	"math/rand/v2"
	"slices"
)

type Holding struct {
	Country string
	Tiles   uint32
}

type World struct {
	Territories []Holding
	Grounds     []Holding
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
		{slot: 1, order: takeFor{flag: flag}, goal: bankGoal, reward: Reward{Kind: RewardShields, Amount: 10}},
		rivalOf(flag, world),
		groundOf(flag, world, random),
	}}
}

func rivalOf(flag string, world World) Mission {
	reward := Reward{Kind: RewardSpreadClicks, Amount: 4}

	var rival Holding
	for _, held := range world.Territories {
		if held.Country != flag && (held.Tiles > rival.Tiles || held.Tiles == rival.Tiles && held.Country < rival.Country) {
			rival = held
		}
	}
	if rival.Tiles < rivalAtLeast {
		return Mission{slot: 2, order: takeFor{flag: flag}, goal: 2 * bankGoal, reward: reward}
	}
	return Mission{slot: 2, order: takeFrom{flag: rival.Country}, goal: rivalGoal, reward: reward}
}

func groundOf(flag string, world World, random *rand.Rand) Mission {
	reward := Reward{Kind: RewardBomb, Amount: 1}

	var grounds []string
	for _, ground := range world.Grounds {
		if ground.Country != flag && ground.Tiles >= groundAtLeast {
			grounds = append(grounds, ground.Country)
		}
	}
	if len(grounds) == 0 {
		return Mission{slot: 3, order: takeFor{flag: flag}, goal: groundGoal, reward: reward}
	}
	slices.Sort(grounds)
	return Mission{slot: 3, order: takeIn{ground: grounds[random.IntN(len(grounds))]}, goal: groundGoal, reward: reward}
}
