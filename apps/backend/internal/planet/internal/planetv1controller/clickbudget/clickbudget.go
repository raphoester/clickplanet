package clickbudget

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Encode sends the bank as it is: a trap sprung leaves it in debt, down to minus one bank, and the page shows the debt.
func Encode(budget clicks.Budget) *planetv1.ClickBudget {
	return &planetv1.ClickBudget{
		Tokens:           budget.Tokens,
		Capacity:         uint32(budget.Capacity),
		RefillPerSecond:  budget.PerSecond,
		Slowdown:         budget.Price.Slowdown,
		Share:            budget.Price.Share,
		NextShare:        budget.Price.NextShare,
		NextSlowdown:     budget.Price.NextSlowdown,
		LinkedMultiplier: budget.LinkedMultiplier,
		SharedWith:       sharedWith[budget.SharedWith],
		Frozen:           bool(budget.Frozen),
		FastStart:        budget.FastStart,
		Spent:            budget.Spent,
	}
}

var sharedWith = map[clicks.SharedWith]planetv1.SharedWith{
	clicks.SharedWithNobody: planetv1.SharedWith_SHARED_WITH_NOBODY,
	clicks.SharedWithGuests: planetv1.SharedWith_SHARED_WITH_GUESTS,
	clicks.SharedWithScope:  planetv1.SharedWith_SHARED_WITH_NETWORK,
}
