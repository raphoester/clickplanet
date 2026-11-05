package bonuses

import "time"

// Never a refill or a bomb: an account made each minute would get a bank or a strike each.
const (
	welcomeBox  = KindSpreadClicks
	welcomeQuiz = KindShields
)

type newcomer struct {
	until time.Time
	boxed bool
	asked bool
}

// An account welcomed keeps the welcome it was given; a new rule applies to the accounts it meets after.
func (r *Registry) welcome(entry *caller, click Click, rules Rules, now time.Time) {
	until := click.Born.Add(rules.Welcome.Within)
	if click.Born.IsZero() || !now.Before(until) {
		return
	}

	account, ok := r.newcomers[click.Holder]
	if !ok {
		account = &newcomer{until: until}
		r.newcomers[click.Holder] = account
	}

	if !account.boxed {
		account.boxed = true
		entry.owedBox = welcomeBox
		entry.nextOfferAt = earliest(entry.nextOfferAt, click.Born.Add(rules.Welcome.BoxAfter))
	}

	if click.Dry && !account.asked && rules.Quiz.Enabled {
		account.asked = true
		entry.owedQuiz = welcomeQuiz
		entry.nextQuizAt = now
	}
}

func (r *Registry) forgetNewcomers(now time.Time) {
	for holder, account := range r.newcomers {
		if !now.Before(account.until) {
			delete(r.newcomers, holder)
		}
	}
}

func earliest(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}

	return a
}
