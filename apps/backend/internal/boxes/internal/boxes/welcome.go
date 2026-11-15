package boxes

import "time"

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
	entry.owedWelcome = true
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
