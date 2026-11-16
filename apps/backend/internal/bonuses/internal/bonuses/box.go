package bonuses

// Never a refill or a bomb: an account made each minute would get a bank or a strike each.
const welcomeBox = KindSpreadClicks

// CaughtBox is a box the boxes module told caught: whom it pays, the token it was offered under, and whether it was a
// newcomer's first box. What it holds is drawn here, at the catch.
type CaughtBox struct {
	Account Account
	Token   string
	Welcome bool
}

// Grant names the payment by the box's token, as it was named while the bonuses module offered the boxes.
func (b CaughtBox) Grant() GrantID {
	return OfferGrantOf(b.Token)
}

// Reward is drawn among the kinds the account may be drawn and the hand has room for; a welcome is a spread whatever the
// gates, or nothing. It is false when there is nothing the box could be worth.
func (b CaughtBox) Reward(rules Rules, full []Kind, unlocked Unlocked) (Reward, bool) {
	if b.Welcome {
		return welcomeReward(rules, full, welcomeBox)
	}
	return drawReward(rules, full, unlocked)
}
