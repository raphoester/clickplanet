package bonuses

import "time"

// Never a refill or a bomb: an account made each minute would get a bank or a strike each.
const welcomeBox = KindSpreadClicks

// An account welcomed keeps the welcome it was given; a new rule applies to the accounts it meets after.
func (r *Registry) welcome(entry *caller, click Click, rules Rules, now time.Time) {
	until := click.Born.Add(rules.Welcome.Within)
	if click.Born.IsZero() || !now.Before(until) {
		return
	}
	if _, boxed := r.newcomers[click.Caller.Account]; boxed {
		return
	}

	r.newcomers[click.Caller.Account] = until
	entry.owedBox = welcomeBox
	entry.nextOfferAt = earliest(entry.nextOfferAt, click.Born.Add(rules.Welcome.BoxAfter))
}

func (r *Registry) forgetNewcomers(now time.Time) {
	for account, until := range r.newcomers {
		if !now.Before(until) {
			delete(r.newcomers, account)
		}
	}
}

func earliest(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}

	return a
}
