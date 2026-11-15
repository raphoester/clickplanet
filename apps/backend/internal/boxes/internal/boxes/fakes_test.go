package boxes

import "time"

// payer is a caller and when its account was made, as these tests click with.
type payer struct {
	Scope   string
	Account Account
	Linked  bool
	Created time.Time
}

func (p payer) caller() Caller {
	return Caller{Scope: p.Scope, Account: p.Account, Linked: p.Linked}
}

func entrantOf(p payer) Entrant {
	return EntrantOf(p.caller())
}

func clickOf(p payer) Click {
	return Click{Caller: p.caller(), Born: p.Created}
}
