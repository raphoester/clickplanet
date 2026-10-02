package titles

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type ID string

type Title interface {
	ID() ID
	Name() string
	EarnedBy(career Career) bool
}

type IDs []ID

func (i IDs) Without(held IDs) IDs {
	return slices.DeleteFunc(slices.Clone(i), func(id ID) bool { return slices.Contains(held, id) })
}

type Holdings map[players.AccountID]IDs

func (h Holdings) Len() int {
	total := 0
	for _, ids := range h {
		total += len(ids)
	}
	return total
}

type Reconciliation struct {
	Grants      Holdings
	Revocations Holdings
}

type Catalog []Title

func (c Catalog) EarnedBy(career Career) IDs {
	if !career.Account.Linked {
		return nil
	}

	var earned IDs
	for _, title := range c {
		if title.EarnedBy(career) {
			earned = append(earned, title.ID())
		}
	}
	return earned
}

func (c Catalog) ReconciliationOf(careers []Career, held Holdings) Reconciliation {
	reconciliation := Reconciliation{Grants: Holdings{}, Revocations: Holdings{}}
	for _, career := range careers {
		account := career.Stats.Account
		earned := c.EarnedBy(career)
		if missing := earned.Without(held[account]); len(missing) > 0 {
			reconciliation.Grants[account] = missing
		}
		if unearned := held[account].Without(earned); len(unearned) > 0 {
			reconciliation.Revocations[account] = unearned
		}
	}
	return reconciliation
}

func (c Catalog) Of(held IDs) []Title {
	var titles []Title
	for _, title := range c {
		if slices.Contains(held, title.ID()) {
			titles = append(titles, title)
		}
	}
	return titles
}
