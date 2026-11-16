package levels

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type ByAccount map[accounts.AccountID]Level

type Reconciliation struct {
	raised  ByAccount
	lowered ByAccount
	removed []accounts.AccountID
}

func (r Reconciliation) Raised() ByAccount { return r.raised }

func (r Reconciliation) Lowered() ByAccount { return r.lowered }

func (r Reconciliation) Removed() []accounts.AccountID { return r.removed }

func (s Scale) ReconciliationOf(
	page []stats.Stats, known map[accounts.AccountID]accounts.Account, kept ByAccount,
) Reconciliation {
	reconciliation := Reconciliation{raised: ByAccount{}, lowered: ByAccount{}}
	for _, counted := range page {
		account := counted.Account()
		standing := s.StandingOf(known[account], counted)
		switch {
		case standing.Empty():
			if _, held := kept[account]; held {
				reconciliation.removed = append(reconciliation.removed, account)
			}
		case standing.Level() > kept[account]:
			reconciliation.raised[account] = standing.Level()
		case standing.Level() < kept[account]:
			reconciliation.lowered[account] = standing.Level()
		}
	}
	return reconciliation
}
