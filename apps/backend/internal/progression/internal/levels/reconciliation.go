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
	reached ByAccount
}

func (r Reconciliation) Raised() ByAccount { return r.raised }

func (r Reconciliation) Lowered() ByAccount { return r.lowered }

func (r Reconciliation) Removed() []accounts.AccountID { return r.removed }

// Reached is every linked account of the page at the level its stats reach, changed or not.
func (r Reconciliation) Reached() ByAccount { return r.reached }

func (s Scale) ReconciliationOf(
	page []stats.Stats, known map[accounts.AccountID]accounts.Account, kept ByAccount,
) Reconciliation {
	reconciliation := Reconciliation{raised: ByAccount{}, lowered: ByAccount{}, reached: ByAccount{}}
	for _, counted := range page {
		account := counted.Account()
		standing := s.StandingOf(known[account], counted)
		if standing.Empty() {
			if _, held := kept[account]; held {
				reconciliation.removed = append(reconciliation.removed, account)
			}
			continue
		}
		reconciliation.reached[account] = standing.Level()
		switch {
		case standing.Level() > kept[account]:
			reconciliation.raised[account] = standing.Level()
		case standing.Level() < kept[account]:
			reconciliation.lowered[account] = standing.Level()
		}
	}
	return reconciliation
}
