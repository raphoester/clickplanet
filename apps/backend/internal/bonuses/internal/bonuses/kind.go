package bonuses

import (
	"crypto/rand"
	"math/big"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Kind string

const (
	KindRefill Kind = "refill"

	KindSpreadClicks Kind = "spread_clicks"

	KindBomb          Kind = "bomb"
	KindEncloseClicks Kind = "enclose_clicks"

	KindShields Kind = "shields"
)

var Kinds = []Kind{KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields}

// Reward is what a box caught or a right answer pays: a kind, and how many of it.
type Reward struct {
	Kind Kind

	Amount int
}

// drawReward draws a kind among those a hand has room for, by their weights, and its amount; a welcome draws its one
// kind or nothing. It is false when there is nothing the win could be worth.
func drawReward(rules Rules, full []Kind, welcome Kind) (Reward, bool) {
	kinds := rules.roomFor(full)
	if welcome != "" {
		kinds = drawable(welcome, kinds)
	}
	if kinds.Empty() {
		return Reward{}, false
	}

	kind := drawKind(kinds, rules.Kinds)
	return Reward{Kind: kind, Amount: amountOf(kind, rules)}, true
}

// roomFor is every kind weighed above 0 but those a hand has no room for.
func (r Rules) roomFor(full []Kind) *cpcolls.Set[Kind] {
	kept := cpcolls.NewSet(full...)
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))
	for _, kind := range Kinds {
		if r.Offered(kind) && !kept.Contains(kind) {
			kinds.Add(kind)
		}
	}

	return kinds
}

func drawable(owed Kind, kinds *cpcolls.Set[Kind]) *cpcolls.Set[Kind] {
	if kinds.Contains(owed) {
		return cpcolls.NewSet(owed)
	}

	return cpcolls.NewSet[Kind]()
}

func drawKind(kinds *cpcolls.Set[Kind], weights map[Kind]float64) Kind {
	total := 0.0
	var last Kind
	for _, kind := range Kinds {
		if kinds.Contains(kind) {
			total += weights[kind]
			last = kind
		}
	}

	const resolution = 1 << 53
	n, err := rand.Int(rand.Reader, big.NewInt(resolution))
	if err != nil {
		return last
	}

	left := float64(n.Int64()) / resolution * total
	for _, kind := range Kinds {
		weight := weights[kind]
		if !kinds.Contains(kind) {
			continue
		}
		if left < weight {
			return kind
		}
		left -= weight
	}

	return last
}

func amountOf(kind Kind, rules Rules) int {
	least, most := 1, 1
	switch kind {
	case KindSpreadClicks:
		most = rules.Spread.MaxPerBox
	case KindEncloseClicks:
		most = rules.Enclose.MaxPerBox
	case KindShields:
		least, most = rules.Shield.MinPerBox, rules.Shield.MaxPerBox
	case KindRefill, KindBomb:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(most-least+1)))
	if err != nil {
		return least
	}

	return least + int(n.Int64())
}
