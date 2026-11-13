package bonuses

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"

// A newcomer's first quiz pays shields: placed with no click, they help a player whose clicks ran out.
const welcomeAnswer = KindShields

// Answer is a right answer the quizzes told: whom it pays, the token it was asked under, and whether it was a
// newcomer's first quiz.
type Answer struct {
	Account Account
	Token   string
	Welcome bool
}

// Grant names the payment by the quiz's token, as it was named while the offers asked the quizzes.
func (a Answer) Grant() GrantID {
	return OfferGrantOf(a.Token)
}

// Reward is drawn as a box's is, among the kinds the hand has room for; a welcome is shields or nothing. It is false
// when there is nothing the answer could be worth.
func (a Answer) Reward(rules Rules, full []Kind) (Reward, bool) {
	kinds := rules.roomFor(full)
	if a.Welcome {
		kinds = drawable(welcomeAnswer, kinds)
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
