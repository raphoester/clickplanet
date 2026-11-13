package quizzes

import "time"

// welcome asks a quiz at once of an account younger than welcome.within whose click left its bank empty, once. An
// account asked keeps it when the rule changes; a new rule applies to the accounts it meets after.
func (r *Registry) welcome(entry *caller, click Click, rules Rules, now time.Time) {
	until := click.Born.Add(rules.Welcome.Within)
	if !click.Dry || !rules.Enabled || click.Born.IsZero() || !now.Before(until) {
		return
	}
	if _, asked := r.newcomers[click.Caller.Account]; asked {
		return
	}

	r.newcomers[click.Caller.Account] = until
	entry.owed = true
	entry.nextAt = now
}

func (r *Registry) forgetNewcomers(now time.Time) {
	for account, until := range r.newcomers {
		if !now.Before(until) {
			delete(r.newcomers, account)
		}
	}
}
