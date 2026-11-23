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

	KindVirus Kind = "virus"

	KindMine Kind = "mine"

	KindRally Kind = "rally"

	KindTrap Kind = "trap"
)

var Kinds = []Kind{
	KindRefill, KindSpreadClicks, KindBomb, KindEncloseClicks, KindShields, KindVirus, KindMine, KindRally, KindTrap,
}

// Reward is what a box caught or a right answer pays: a kind, and how many of it.
type Reward struct {
	Kind Kind

	Amount int
}

// drawReward draws a kind among those the account may be drawn, by their weights, and its amount, whatever the hand
// holds: the wallet keeps what fits, so holding every other kind full never makes the draw any surer. It is false when
// the kinds it may be drawn all weigh 0.
func drawReward(rules Rules, unlocked Unlocked) (Reward, bool) {
	kinds := rules.drawable(unlocked)
	if kinds.Empty() {
		return Reward{}, false
	}

	kind := drawKind(kinds, rules.Kinds)
	return Reward{Kind: kind, Amount: amountOf(kind, rules)}, true
}

// drawable is every kind weighed above 0 that the account may be drawn.
func (r Rules) drawable(unlocked Unlocked) *cpcolls.Set[Kind] {
	kinds := cpcolls.NewSetWithCapacity[Kind](len(Kinds))
	for _, kind := range Kinds {
		if r.Offered(kind) && r.MayDraw(kind, unlocked) {
			kinds.Add(kind)
		}
	}

	return kinds
}

// welcomeReward is its one kind whatever the gates, since a newcomer is a guest: false when the set weighs it 0.
func welcomeReward(rules Rules, kind Kind) (Reward, bool) {
	if !rules.Offered(kind) {
		return Reward{}, false
	}

	return Reward{Kind: kind, Amount: amountOf(kind, rules)}, true
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
	case KindMine:
		most = rules.Mine.MaxPerBox
	case KindRally:
		most = rules.Rally.MaxPerBox
	case KindTrap:
		most = rules.Trap.MaxPerBox
	case KindRefill, KindBomb, KindVirus:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(most-least+1)))
	if err != nil {
		return least
	}

	return least + int(n.Int64())
}
