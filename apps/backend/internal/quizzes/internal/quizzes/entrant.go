package quizzes

// Entrant is who a schedule is kept for: the account a linked token names, or the scope every guest behind it shares.
type Entrant string

func EntrantOf(caller Caller) Entrant {
	if caller.Linked {
		return Entrant("account:" + string(caller.Account))
	}

	return Entrant(caller.Scope)
}
