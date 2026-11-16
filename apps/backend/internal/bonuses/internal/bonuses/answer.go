package bonuses

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

// Reward is drawn among the kinds the account may be drawn and the hand has room for; a welcome is shields whatever the
// gates, or nothing. It is false when there is nothing the answer could be worth.
func (a Answer) Reward(rules Rules, full []Kind, unlocked Unlocked) (Reward, bool) {
	if a.Welcome {
		return welcomeReward(rules, full, welcomeAnswer)
	}
	return drawReward(rules, full, unlocked)
}
