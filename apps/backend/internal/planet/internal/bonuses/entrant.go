package bonuses

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Entrant string

func EntrantOf(payer clicks.Payer) Entrant {
	if payer.Linked {
		return Entrant("account:" + payer.Account)
	}

	return Entrant(payer.Scope)
}

func ClickOf(payer clicks.Payer, dry bool) Click {
	return Click{
		Entrant: EntrantOf(payer),
		Scope:   payer.Scope,
		Holder:  HolderOf(payer),
		Born:    payer.Created,
		Dry:     dry,
	}
}
